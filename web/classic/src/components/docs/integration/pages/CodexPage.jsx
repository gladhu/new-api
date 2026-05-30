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

import React from 'react';
import { Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import DocCallout from '../DocCallout';
import DocCodeBlock, { DocInlineCode } from '../DocCodeBlock';
import { DocPageHeader, DocSection, DocStepList, DocBulletList } from '../DocSection';
import { FACEAPI_BASE_URL } from '../constants';

const CODEX_INSTALL = 'npm install -g @openai/codex';

const CODEX_RELAY_INSTALL = `python3 -m venv ~/.codex/venv
~/.codex/venv/bin/pip install -U codex-relay`;

const CODEX_AUTH = `{
  "FACEAPI_API_KEY": "sk-xxxx"
}`;

const CODEX_CONFIG = `model = "gpt-5.1"
model_provider = "facecloud-relay"

[tools.tool_search]
enabled = false

[model_providers.facecloud-relay]
name = "FaceCloud (via codex-relay)"
base_url = "http://127.0.0.1:4444/v1"
env_key = "FACEAPI_API_KEY"
wire_api = "responses"

[model_properties."gpt-5.1"]
context_window = 272000
max_context_window = 1000000
supports_parallel_tool_calls = true
supports_reasoning_summaries = false
input_modalities = ["text"]`;

const CODEX_START_RELAY = `#!/bin/zsh
set -euo pipefail

CODEX_HOME="\${CODEX_HOME:-$HOME/.codex}"
VENV_BIN="$CODEX_HOME/venv/bin/codex-relay"
AUTH_FILE="$CODEX_HOME/auth.json"

API_KEY="$(python3 -c \\"import json; print(json.load(open('$AUTH_FILE'))['FACEAPI_API_KEY'])\\")"

export CODEX_RELAY_UPSTREAM="${FACEAPI_BASE_URL}/v1"
export CODEX_RELAY_API_KEY="$API_KEY"
export CODEX_RELAY_PORT="\${CODEX_RELAY_PORT:-4444}"

exec "$VENV_BIN"`;

const CODEX_RELAY_AUTOSTART = `# Optional: auto-start relay when opening a shell
if ! lsof -i :4444 -sTCP:LISTEN >/dev/null 2>&1; then
  nohup ~/.codex/start-relay.sh > ~/.codex/relay.log 2>&1 &
fi`;

const CodexPage = () => {
  const { t } = useTranslation();

  return (
    <article>
      <DocPageHeader
        title={t('Codex')}
        description={t(
          'Configure OpenAI Codex CLI to use FaceCloud through codex-relay (recommended).',
        )}
      />

      <DocCallout title={t('Recommended')}>
        {t(
          'We recommend running codex-relay locally: Codex speaks the Responses API, while FaceCloud exposes Chat Completions. codex-relay translates between them and avoids errors such as missing tools[N].tools.',
        )}
      </DocCallout>

      <DocSection title={t('Why codex-relay?')}>
        <Typography.Paragraph type='secondary' style={{ lineHeight: 1.7, marginBottom: '16px' }}>
          {t(
            'Codex CLI v0.130+ uses the OpenAI Responses API. Most gateways only expose Chat Completions. codex-relay translates between the two on your machine before forwarding requests to FaceCloud.',
          )}
        </Typography.Paragraph>
        <DocBulletList
          items={[
            t('Matches Codex native protocol (wire_api = responses)'),
            t('Translates tool definitions and streaming events for gateway compatibility'),
            t('Works with standard FaceCloud API keys — no server-side Codex channel required'),
          ]}
        />
      </DocSection>

      <DocSection title={t('Install Codex CLI')}>
        <DocCodeBlock code={CODEX_INSTALL} />
      </DocSection>

      <DocSection title={t('Install codex-relay')}>
        <Typography.Paragraph type='secondary' style={{ lineHeight: 1.7, marginBottom: '16px' }}>
          {t('Create a dedicated Python virtual environment and install codex-relay:')}
        </Typography.Paragraph>
        <DocCodeBlock code={CODEX_RELAY_INSTALL} />
      </DocSection>

      <DocSection title={t('Set your API key')}>
        <Typography.Paragraph type='secondary' style={{ lineHeight: 1.7, marginBottom: '16px' }}>
          {t('Store your FaceCloud API key in ~/.codex/auth.json:')}
        </Typography.Paragraph>
        <DocCodeBlock code={CODEX_AUTH} filename='~/.codex/auth.json' />
        <Typography.Paragraph type='secondary' size='small' style={{ marginTop: '8px' }}>
          {t('Replace')} <DocInlineCode>sk-xxxx</DocInlineCode> {t('with your FaceCloud API key.')}
        </Typography.Paragraph>
      </DocSection>

      <DocSection title={t('Configure Codex')}>
        <DocStepList
          steps={[
            <>
              <p>
                {t('Create or edit')} <DocInlineCode>~/.codex/config.toml</DocInlineCode>.
              </p>
            </>,
            <>
              <p>
                {t(
                  'Point Codex at the local relay (not FaceCloud directly). The relay forwards to FaceCloud using Chat Completions:',
                )}
              </p>
              <DocCodeBlock code={CODEX_CONFIG} filename='~/.codex/config.toml' />
            </>,
            <>
              <p>
                {t('Replace')} <DocInlineCode>gpt-5.1</DocInlineCode>{' '}
                {t('with a model enabled on your FaceCloud account.')}
              </p>
            </>,
          ]}
        />
      </DocSection>

      <DocSection title={t('Start codex-relay')}>
        <Typography.Paragraph type='secondary' style={{ lineHeight: 1.7, marginBottom: '16px' }}>
          {t('Before using Codex, start the relay in a separate terminal:')}
        </Typography.Paragraph>
        <DocStepList
          steps={[
            <>
              <p>
                {t('Save the script as')}{' '}
                <DocInlineCode>~/.codex/start-relay.sh</DocInlineCode> {t('and make it executable:')}
              </p>
              <DocCodeBlock code={CODEX_START_RELAY} filename='~/.codex/start-relay.sh' />
              <DocCodeBlock code='chmod +x ~/.codex/start-relay.sh' />
            </>,
            <>
              <p>{t('Run the relay and keep the terminal open, or use the optional background snippet below.')}</p>
              <DocCodeBlock code='~/.codex/start-relay.sh' />
            </>,
          ]}
        />
        <Typography.Paragraph strong style={{ marginTop: '16px', marginBottom: '8px' }}>
          {t('Optional: keep relay running in the background')}
        </Typography.Paragraph>
        <DocCodeBlock code={CODEX_RELAY_AUTOSTART} filename='~/.zshrc or ~/.bashrc' />
      </DocSection>

      <DocSection title={t('Run Codex')}>
        <DocStepList
          steps={[
            <p key='relay'>{t('Ensure codex-relay is listening on http://127.0.0.1:4444.')}</p>,
            <p key='codex'>{t('In another terminal, run codex and send a test prompt.')}</p>,
            <p key='verify'>{t('Run codex doctor to verify auth and local relay connectivity.')}</p>,
          ]}
        />
        <DocCodeBlock code='codex doctor\ncodex' />
      </DocSection>

      <DocSection title={t('Configuration reference')}>
        <DocBulletList
          items={[
            <>
              <DocInlineCode>base_url</DocInlineCode> —{' '}
              {t('Local relay listen address for Codex (wire_api = responses)')}
            </>,
            <>
              <DocInlineCode>CODEX_RELAY_UPSTREAM</DocInlineCode> —{' '}
              {t('Upstream FaceCloud endpoint used by codex-relay: {{url}}', {
                url: `${FACEAPI_BASE_URL}/v1`,
              })}
            </>,
            <>
              <DocInlineCode>env_key</DocInlineCode> — {t('Environment variable holding your API key')}
            </>,
            <>
              <DocInlineCode>wire_api</DocInlineCode> — {t('Use Responses API wire format (Codex relay)')}
            </>,
            <>
              <DocInlineCode>[tools.tool_search] enabled = false</DocInlineCode> —{' '}
              {t('Disables tool_search to avoid incompatible nested tool definitions on some gateways')}
            </>,
          ]}
        />
      </DocSection>

      <DocCallout variant='warning' title={t('Tip')} className='mt-6'>
        {t(
          'If you see errors like missing tools[N].tools, ensure Codex points at http://127.0.0.1:4444/v1 and codex-relay is running — do not point Codex directly at the FaceCloud gateway.',
        )}
      </DocCallout>
    </article>
  );
};

export default CodexPage;
