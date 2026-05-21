import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import App from './App';

describe('创作中心 mock H5', () => {
  it('从小说列表进入流程总览，不直接进入聊天', async () => {
    const user = userEvent.setup();
    render(<App />);

    expect(screen.getByRole('heading', { name: '创作中心' })).toBeInTheDocument();
    await user.click(screen.getAllByRole('button', { name: '进入流程' })[0]);

    expect(screen.getByText('小说创作流程')).toBeInTheDocument();
    expect(screen.queryByText('AI 聊天')).not.toBeInTheDocument();
  });

  it('阻止进入未解锁步骤并展示中文原因', async () => {
    const user = userEvent.setup();
    render(<App />);

    await user.click(screen.getAllByRole('button', { name: '进入流程' })[1]);
    await user.click(screen.getByRole('button', { name: /剧情线/ }));

    expect(screen.getByText('请先归档小说大纲，再开始剧情线。')).toBeInTheDocument();
  });

  it('归档当前步骤后解锁下一步骤', async () => {
    const user = userEvent.setup();
    render(<App />);

    await user.click(screen.getAllByRole('button', { name: '进入流程' })[1]);
    await user.click(screen.getByRole('button', { name: /小说大纲/ }));
    await user.click(screen.getByRole('button', { name: '归档大纲' }));
    await user.click(screen.getByRole('button', { name: '确认归档' }));

    const plotlineStep = screen.getByRole('button', { name: /剧情线/ });
    expect(within(plotlineStep).getByText('当前步骤 · 继续创作')).toBeInTheDocument();
  });

  it('修改已归档内容前展示影响下游的警告', async () => {
    const user = userEvent.setup();
    render(<App />);

    await user.click(screen.getAllByRole('button', { name: '进入流程' })[0]);
    await user.click(screen.getByRole('button', { name: /小说大纲/ }));
    await user.click(screen.getByRole('button', { name: '修改归档' }));

    expect(screen.getByText('修改已归档内容？')).toBeInTheDocument();
    expect(screen.getByText(/修改归档内容会影响下面步骤已经创作的内容/)).toBeInTheDocument();
  });

  it('语音输入结果作为文字建议展示在输入框', async () => {
    const user = userEvent.setup();
    render(<App />);

    await user.click(screen.getAllByRole('button', { name: '进入流程' })[0]);
    await user.click(screen.getByRole('button', { name: /关键人物设定/ }));
    await user.click(screen.getByRole('button', { name: '语音输入' }));

    expect(screen.getByDisplayValue('帮我增加一个亦敌亦友角色')).toBeInTheDocument();
  });

  it('个人中心展示占位信息', async () => {
    const user = userEvent.setup();
    render(<App />);

    await user.click(screen.getByRole('button', { name: '个人' }));

    expect(screen.getByRole('heading', { name: '个人中心' })).toBeInTheDocument();
    expect(screen.getByText('创作者昵称')).toBeInTheDocument();
  });
});
