/*
Copyright (C) 2025

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

import React, { useEffect, useState } from 'react';
import {
  Button,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Typography,
} from '@douyinfe/semi-ui';
import { downloadAdminUserLogExport, showError, showSuccess } from '../../../../helpers';

const { Text } = Typography;

const MONTH_OPTIONS = Array.from({ length: 12 }, (_, i) => ({
  label: String(i + 1),
  value: i + 1,
}));

const UserBillExportModal = ({ visible, onCancel, user, t }) => {
  const now = new Date();
  const [year, setYear] = useState(now.getFullYear());
  const [month, setMonth] = useState(now.getMonth() + 1);
  const [timezone, setTimezone] = useState('');
  const [loadingBill, setLoadingBill] = useState(false);
  const [loadingDetails, setLoadingDetails] = useState(false);
  const [loadingAll, setLoadingAll] = useState(false);

  useEffect(() => {
    if (!visible) return;
    const d = new Date();
    setYear(d.getFullYear());
    setMonth(d.getMonth() + 1);
    setTimezone('');
  }, [visible]);

  if (!user) return null;

  const run = async (kind, setBusy) => {
    const y = Number(year);
    const m = Number(month);
    if (!Number.isFinite(y) || y < 1970 || y > 9999) {
      showError(t('年份无效'));
      return;
    }
    if (!Number.isFinite(m) || m < 1 || m > 12) {
      showError(t('月份无效'));
      return;
    }
    setBusy(true);
    try {
      await downloadAdminUserLogExport(kind, {
        userId: user.id,
        year: y,
        month: m,
        timezone: timezone.trim() || undefined,
      });
      showSuccess(t('已开始下载'));
      onCancel();
    } catch (e) {
      showError(e?.message || t('导出失败'));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal
      title={t('导出用量CSV')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      width={480}
    >
      <Space vertical align='start' spacing='loose' style={{ width: '100%' }}>
        <Text type='secondary'>
          {t('用量导出说明', {
            name: user.username,
            id: user.id,
          })}
        </Text>
        <div style={{ width: '100%' }}>
          <Text strong>{t('年份')}</Text>
          <InputNumber
            style={{ width: '100%', marginTop: 8 }}
            min={1970}
            max={9999}
            value={year}
            onChange={(v) => setYear(v ?? new Date().getFullYear())}
          />
        </div>
        <div style={{ width: '100%' }}>
          <Text strong>{t('月份')}</Text>
          <Select
            style={{ width: '100%', marginTop: 8 }}
            optionList={MONTH_OPTIONS}
            value={month}
            onChange={(v) => setMonth(v)}
          />
        </div>
        <div style={{ width: '100%' }}>
          <Text strong>{t('时区（IANA，可选）')}</Text>
          <Input
            style={{ marginTop: 8 }}
            value={timezone}
            onChange={(v) => setTimezone(v)}
            placeholder={t('例如 Asia/Shanghai')}
          />
          <Text type='tertiary' style={{ marginTop: 6, display: 'block', fontSize: 12 }}>
            {t('用量导出时区说明')}
          </Text>
        </div>
        <Space wrap style={{ marginTop: 8 }}>
          <Button
            loading={loadingBill}
            disabled={loadingDetails || loadingAll}
            onClick={() => run('monthly_bill', setLoadingBill)}
          >
            {t('导出月账单')}
          </Button>
          <Button
            type='primary'
            loading={loadingDetails}
            disabled={loadingBill || loadingAll}
            onClick={() => run('consumption_details', setLoadingDetails)}
          >
            {t('导出消费明细')}
          </Button>
          <Button
            type='primary'
            loading={loadingAll}
            disabled={loadingBill || loadingDetails}
            onClick={() =>
              run('monthly_bill_and_consumption_details', setLoadingAll)
            }
          >
            {t('导出月账单和消费明细')}
          </Button>
        </Space>
      </Space>
    </Modal>
  );
};

export default UserBillExportModal;
