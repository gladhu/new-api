/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  DatePicker,
  Descriptions,
  Input,
  Modal,
  Space,
  Spin,
  Typography,
} from '@douyinfe/semi-ui';
import dayjs from 'dayjs';
import { API, showError } from '../../../../helpers';
import { renderQuota } from '../../../../helpers/render';
import { DATE_RANGE_PRESETS } from '../../../../constants/console.constants';

const defaultDateRange = () => [
  dayjs().startOf('month').toDate(),
  dayjs().endOf('day').toDate(),
];

const datePresets = DATE_RANGE_PRESETS.map((preset) => ({
  text: preset.text,
  start: preset.start(),
  end: preset.end(),
}));

function ChannelConsumptionContent({ t, record }) {
  const [dateRange, setDateRange] = useState(defaultDateRange);
  const [userIdInput, setUserIdInput] = useState('');
  const [usernameInput, setUsernameInput] = useState('');
  const [appliedUserId, setAppliedUserId] = useState(null);
  const [appliedUsername, setAppliedUsername] = useState('');
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState(null);

  const userFilterActive = appliedUserId != null || !!appliedUsername;

  const fetchConsumption = useCallback(async () => {
    if (!record?.id) return;
    const start = dateRange?.[0];
    const end = dateRange?.[1];
    if (!start || !end) {
      showError(t('请选择时间范围'));
      return;
    }

    const params = {
      start_timestamp: Math.floor(new Date(start).getTime() / 1000),
      end_timestamp: Math.floor(new Date(end).getTime() / 1000),
    };
    if (appliedUserId) {
      params.user_id = appliedUserId;
    } else if (appliedUsername) {
      params.username = appliedUsername;
    }

    setLoading(true);
    try {
      const res = await API.get(`/api/channel/${record.id}/consumption`, {
        params,
      });
      const { success, message, data: payload } = res.data;
      if (!success) {
        throw new Error(message || t('加载消费统计失败'));
      }
      setData(payload);
    } catch (error) {
      showError(error?.message || String(error));
      setData(null);
    } finally {
      setLoading(false);
    }
  }, [appliedUserId, appliedUsername, dateRange, record?.id, t]);

  useEffect(() => {
    fetchConsumption();
  }, [fetchConsumption]);

  const applyUserFilter = () => {
    const trimmedUsername = usernameInput.trim();
    const parsedUserId = Number.parseInt(userIdInput.trim(), 10);
    if (userIdInput.trim()) {
      if (!Number.isFinite(parsedUserId) || parsedUserId <= 0) {
        showError(t('用户ID无效'));
        return;
      }
      setAppliedUserId(parsedUserId);
      setAppliedUsername('');
    } else {
      setAppliedUserId(null);
      setAppliedUsername(trimmedUsername);
    }
  };

  const clearUserFilter = () => {
    setUserIdInput('');
    setUsernameInput('');
    setAppliedUserId(null);
    setAppliedUsername('');
  };

  const totalTokens = useMemo(() => {
    if (!data) return 0;
    return (
      Number(data.prompt_tokens || 0) + Number(data.completion_tokens || 0)
    );
  }, [data]);

  return (
    <div className='flex flex-col gap-4'>
      <Typography.Text type='tertiary' size='small'>
        {record?.name} (#{record?.id})
      </Typography.Text>

      <div>
        <Typography.Text strong size='small' className='mb-2 block'>
          {t('时间范围')}
        </Typography.Text>
        <DatePicker
          type='dateTimeRange'
          className='w-full'
          value={dateRange}
          onChange={(value) => setDateRange(value)}
          presets={datePresets.map((preset) => ({
            text: t(preset.text),
            start: preset.start,
            end: preset.end,
          }))}
        />
      </div>

      <div className='rounded-lg border border-[var(--semi-color-border)] p-3'>
        <Typography.Text strong size='small' className='mb-2 block'>
          {t('按用户筛选（可选）')}
        </Typography.Text>
        <Space wrap className='w-full'>
          <Input
            prefix={t('用户ID')}
            value={userIdInput}
            onChange={setUserIdInput}
            placeholder='123'
            style={{ width: 140 }}
            size='small'
          />
          <Input
            prefix={t('用户名')}
            value={usernameInput}
            onChange={setUsernameInput}
            placeholder='alice'
            style={{ width: 160 }}
            size='small'
          />
          <Button size='small' onClick={applyUserFilter}>
            {t('应用用户筛选')}
          </Button>
          {userFilterActive ? (
            <Button size='small' type='tertiary' onClick={clearUserFilter}>
              {t('清除用户筛选')}
            </Button>
          ) : null}
        </Space>
        {userFilterActive ? (
          <Typography.Text type='tertiary' size='small' className='mt-2 block'>
            {appliedUserId != null
              ? `${t('用户ID')}: ${appliedUserId}`
              : `${t('用户名')}: ${appliedUsername}`}
          </Typography.Text>
        ) : null}
      </div>

      <Space>
        <Button
          theme='solid'
          type='primary'
          size='small'
          loading={loading}
          onClick={fetchConsumption}
        >
          {t('查询')}
        </Button>
      </Space>

      <Spin spinning={loading}>
        {data ? (
          <Descriptions
            row
            size='small'
            data={[
              {
                key: 'usage',
                label: t('消耗额度'),
                value: renderQuota(Number(data.quota || 0)),
              },
              {
                key: 'requests',
                label: t('请求总数'),
                value: String(data.request_count ?? 0),
              },
              {
                key: 'tokens',
                label: t('Token 总数'),
                value: String(totalTokens),
              },
              ...(!userFilterActive
                ? [
                    {
                      key: 'lifetime',
                      label: t('渠道历史总消耗'),
                      value: renderQuota(Number(data.lifetime_used_quota || 0)),
                    },
                  ]
                : []),
            ]}
          />
        ) : (
          !loading && (
            <Typography.Text type='tertiary'>{t('暂无数据')}</Typography.Text>
          )
        )}
      </Spin>
    </div>
  );
}

export function openChannelConsumptionModal({ t, record }) {
  const tt = typeof t === 'function' ? t : (v) => v;

  Modal.info({
    title: tt('渠道消费统计'),
    width: 560,
    centered: true,
    content: <ChannelConsumptionContent t={tt} record={record} />,
    footer: (
      <div className='flex justify-end'>
        <Button type='primary' theme='solid' onClick={() => Modal.destroyAll()}>
          {tt('关闭')}
        </Button>
      </div>
    ),
  });
}