import { useEffect, useMemo, useState } from 'react';
import { Archive, ChevronLeft, Lock, Mic, Plus, Send, UserRound } from 'lucide-react';

type Tab = 'creation' | 'profile';
type Screen = 'list' | 'workflow' | 'step';
type StepStatus = 'locked' | 'active' | 'archived' | 'review';
type StepKey = 'outline' | 'plotlines' | 'characters' | 'volumes' | 'chapters' | 'content';

type StepConfig = {
  key: StepKey;
  name: string;
  shortName: string;
  archiveText: string;
  inputPlaceholder: string;
  dependsOn?: StepKey;
  upstreamTitle: string;
  sampleContent: string[];
  aiTip: string;
};

type Novel = {
  id: number;
  title: string;
  genre: string;
  updatedAt: string;
  archivedCount: number;
  reviewFrom?: StepKey;
};

type StepContent = {
  lines: string[];
  upstreamSummary: string;
  aiTip: string;
};

type Profile = {
  id: number;
  nickname: string;
  avatar_url: string;
  bio: string;
};

type ProfileStats = {
  novel_count: number;
  word_count: number;
};

type NovelApiItem = {
  id: number;
  title: string;
  genre: string;
  updated_at: string;
  archived_count: number;
};

const apiBase = '/api/h5';

const steps: StepConfig[] = [
  {
    key: 'outline',
    name: '小说大纲',
    shortName: '大纲',
    archiveText: '归档大纲',
    inputPlaceholder: '输入你的大纲建议...',
    upstreamTitle: '无上游依赖',
    sampleContent: ['标题：麒麟神座：万界商途', '世界观：万界以商道、灵脉和契约相互连通。', '核心冲突：主角要在商盟、宗门和异族之间守住万界商令。'],
    aiTip: 'AI：核心冲突已经明确，可以补强反派动机。',
  },
  {
    key: 'plotlines',
    name: '剧情线',
    shortName: '剧情线',
    archiveText: '归档剧情线',
    inputPlaceholder: '输入剧情线建议...',
    dependsOn: 'outline',
    upstreamTitle: '上游摘要：小说大纲',
    sampleContent: ['主线｜万界商途崛起：主角从暗市交易开始进入万界商盟。', '支线｜宗门旧怨：旧债在第二卷集中爆发。'],
    aiTip: 'AI：支线与主线转折需要在第一卷末建立钩子。',
  },
  {
    key: 'characters',
    name: '关键人物设定',
    shortName: '关键人物',
    archiveText: '归档人物设定',
    inputPlaceholder: '输入人物建议...',
    dependsOn: 'plotlines',
    upstreamTitle: '上游摘要：大纲 + 剧情线',
    sampleContent: ['主角｜陆玄：沉稳、冒险、重承诺，目标是掌控万界商路。', '反派｜沈夜：隐忍、多疑，想夺取万界商令。'],
    aiTip: 'AI：建议给沈夜增加与主角相似但相反的价值观。',
  },
  {
    key: 'volumes',
    name: '分卷',
    shortName: '分卷',
    archiveText: '归档分卷',
    inputPlaceholder: '输入分卷建议...',
    dependsOn: 'characters',
    upstreamTitle: '上游摘要：大纲 / 剧情线 / 人物',
    sampleContent: ['第一卷：万界商令｜获得商令与入局。', '第二卷：宗门旧债｜支线冲突集中爆发。'],
    aiTip: 'AI：第二卷可以承接第一卷暗市伏笔，避免割裂。',
  },
  {
    key: 'chapters',
    name: '章节划分',
    shortName: '章节',
    archiveText: '归档章节划分',
    inputPlaceholder: '输入章节建议...',
    dependsOn: 'volumes',
    upstreamTitle: '上游摘要：当前分卷',
    sampleContent: ['第 1 章 商令初现：主角获得万界商令，暗市交易失败。', '第 2 章 暗市风波：第一次交易冲突升级。'],
    aiTip: 'AI：章节号连续，建议第 3 章安排一次小失败。',
  },
  {
    key: 'content',
    name: '正文撰写',
    shortName: '正文',
    archiveText: '归档正文',
    inputPlaceholder: '输入正文建议...',
    dependsOn: 'chapters',
    upstreamTitle: '章节规划',
    sampleContent: ['夜色压在暗市低矮的檐角。', '陆玄握紧掌心的铜令，第一次听见万界商路的回响。'],
    aiTip: 'AI：正文承接清晰，可以增加主角失败后的心理描写。',
  },
];

const initialNovels: Novel[] = [
  { id: 1, title: '麒麟神座：万界商途', genre: '玄幻', updatedAt: '今天 14:20', archivedCount: 2 },
  { id: 2, title: '长夜之后', genre: '科幻', updatedAt: '昨天 21:03', archivedCount: 0 },
];

function getStepStatus(novel: Novel, index: number): StepStatus {
  if (novel.reviewFrom && index > steps.findIndex((step) => step.key === novel.reviewFrom)) {
    return 'review';
  }
  if (index < novel.archivedCount) return 'archived';
  if (index === novel.archivedCount) return 'active';
  return 'locked';
}

export default function App() {
  const [tab, setTab] = useState<Tab>('creation');
  const [screen, setScreen] = useState<Screen>('list');
  const [novels, setNovels] = useState(initialNovels);
  const [selectedNovelId, setSelectedNovelId] = useState<number | null>(null);
  const [selectedStepKey, setSelectedStepKey] = useState<StepKey | null>(null);
  const [toast, setToast] = useState('');
  const [archiveDialog, setArchiveDialog] = useState(false);
  const [editDialog, setEditDialog] = useState(false);
  const [suggestion, setSuggestion] = useState('');
  const [stepContent, setStepContent] = useState<StepContent | null>(null);
  const [loading, setLoading] = useState(false);
  const [profile, setProfile] = useState<Profile | null>(null);
  const [profileStats, setProfileStats] = useState<ProfileStats | null>(null);

  useEffect(() => {
    // 首屏进入创作中心时主动拉取小说列表；接口失败时保留本地示例，避免 H5 白屏影响联调。
    void loadNovels();
  }, []);

  useEffect(() => {
    // 个人中心只有在用户切换到该 Tab 时才读取，减少创作页首屏对下游接口的并发请求压力。
    if (tab === 'profile') {
      void loadProfile();
    }
  }, [tab]);

  const selectedNovel = useMemo(
    () => novels.find((novel) => novel.id === selectedNovelId) ?? null,
    [novels, selectedNovelId],
  );
  const selectedStep = steps.find((step) => step.key === selectedStepKey) ?? null;

  const loadNovels = async () => {
    try {
      const payload = await apiGet<{ novels: NovelApiItem[] }>('/novels');
      setNovels(payload.novels.map(normalizeNovel));
    } catch (error) {
      setToast(getErrorMessage(error, '后端暂未连接，当前展示本地示例数据。'));
    }
  };

  const loadProfile = async () => {
    try {
      // 个人资料与统计来自两个后端接口；并发读取可保持 Tab 切换后的响应速度。
      const [me, stats] = await Promise.all([apiGet<Profile>('/me'), apiGet<ProfileStats>('/me/stats')]);
      setProfile(me);
      setProfileStats(stats);
    } catch (error) {
      setToast(getErrorMessage(error, '个人中心接口暂不可用，已展示默认信息。'));
    }
  };

  const createNovel = async () => {
    const title = window.prompt('请输入小说标题');
    if (!title?.trim()) return;
    const genre = window.prompt('请输入小说类型', '玄幻') ?? '';
    const originalOutline = window.prompt('请输入初始构思，可稍后在大纲步骤补充', '') ?? '';
    try {
      // 创建接口返回完整小说记录；前端只抽取列表所需字段，避免把后端模型细节扩散到 UI 状态。
      const payload = await apiPost<{ novel: { id: number; title: string; genre: string; updated_at?: string; UpdatedAt?: string }; workflow?: { archived_count?: number } }>('/novels', {
        title: title.trim(),
        genre: genre.trim(),
        original_outline: originalOutline.trim(),
      });
      const created = normalizeNovel({
        id: payload.novel.id,
        title: payload.novel.title,
        genre: payload.novel.genre,
        updated_at: payload.novel.updated_at ?? payload.novel.UpdatedAt ?? new Date().toISOString(),
        archived_count: payload.workflow?.archived_count ?? 0,
      });
      setNovels((current) => [created, ...current]);
      setToast('小说创建成功，可以进入流程完善大纲。');
    } catch (error) {
      setToast(getErrorMessage(error, '创建小说失败，请检查后端服务是否启动。'));
    }
  };

  const openWorkflow = async (novelId: number) => {
    setSelectedNovelId(novelId);
    setScreen('workflow');
    setToast('');
    try {
      const payload = await apiGet<{ archived_count: number }>(`/novels/${novelId}/workflow`);
      setNovels((current) => current.map((novel) => (novel.id === novelId ? { ...novel, archivedCount: payload.archived_count } : novel)));
    } catch (error) {
      setToast(getErrorMessage(error, '流程状态读取失败，已使用当前缓存状态。'));
    }
  };

  const openStep = async (step: StepConfig) => {
    if (!selectedNovel) return;
    const index = steps.findIndex((item) => item.key === step.key);
    const status = getStepStatus(selectedNovel, index);
    if (status === 'locked') {
      const previous = steps[index - 1];
      setToast(`请先归档${previous.name}，再开始${step.name}。`);
      return;
    }
    setSelectedStepKey(step.key);
    setScreen('step');
    setToast('');
    setLoading(true);
    try {
      const payload = await apiGet<Record<string, unknown>>(`/novels/${selectedNovel.id}/steps/${step.key}`);
      setStepContent(normalizeStepContent(step, payload));
    } catch (error) {
      setStepContent({ lines: step.sampleContent, upstreamSummary: '接口读取失败，暂时展示本地示例内容。', aiTip: step.aiTip });
      setToast(getErrorMessage(error, '接口读取失败，暂时展示本地示例内容。'));
    } finally {
      setLoading(false);
    }
  };

  const confirmArchive = async () => {
    if (!selectedNovel || !selectedStep) return;
    const index = steps.findIndex((step) => step.key === selectedStep.key);
    setNovels((current) =>
      current.map((novel) =>
        novel.id === selectedNovel.id
          ? { ...novel, archivedCount: Math.max(novel.archivedCount, index + 1), reviewFrom: undefined }
          : novel,
      ),
    );
    try {
      await apiPost(`/novels/${selectedNovel.id}/steps/${selectedStep.key}/archive`, { user_suggestion: suggestion });
      setToast('归档成功，下一步骤已解锁。');
    } catch (error) {
      setToast(getErrorMessage(error, '归档接口暂不可用，已先更新本地流程状态。'));
    } finally {
      setArchiveDialog(false);
      setScreen('workflow');
    }
  };

  const confirmEditArchived = async () => {
    if (!selectedNovel || !selectedStep) return;
    setNovels((current) =>
      current.map((novel) => (novel.id === selectedNovel.id ? { ...novel, reviewFrom: selectedStep.key } : novel)),
    );
    try {
      await apiPost(`/novels/${selectedNovel.id}/steps/${selectedStep.key}/edit-archived`, {});
    } catch {
      // 修改归档的确认已经由弹窗完成，接口失败时仍允许前端进入风险编辑态，避免阻断用户整理草稿。
    } finally {
      setEditDialog(false);
      setToast('正在修改归档内容，下游步骤可能需要重新检查。');
    }
  };

  const useVoiceInput = async () => {
    try {
      const payload = await apiPost<{ text: string }>('/speech/transcriptions', {});
      setSuggestion(payload.text);
    } catch {
      setSuggestion('帮我增加一个亦敌亦友角色');
    }
  };

  const sendSuggestion = async () => {
    if (!selectedNovel || !selectedStep || !suggestion.trim()) return;
    try {
      await apiPost(`/novels/${selectedNovel.id}/steps/${selectedStep.key}/messages`, { user_suggestion: suggestion.trim() });
      setToast('建议已发送，AI 会围绕当前步骤处理。');
      setSuggestion('');
    } catch (error) {
      setToast(getErrorMessage(error, '建议发送失败，请检查后端服务是否启动。'));
    }
  };

  const saveDraft = async () => {
    if (!selectedNovel || !selectedStep) return;
    try {
      await apiPut(`/novels/${selectedNovel.id}/steps/${selectedStep.key}/draft`, { content: { suggestion, lines: stepContent?.lines ?? selectedStep.sampleContent } });
      setToast('草稿已保存，不会解锁下一步骤。');
    } catch (error) {
      setToast(getErrorMessage(error, '当前步骤草稿保存需要使用对应工作流生成接口。'));
    }
  };

  const deleteNovel = async (novelId: number) => {
    if (!window.confirm('删除后不可恢复，确认删除这部小说吗？')) return;
    setNovels((current) => current.filter((novel) => novel.id !== novelId));
    try {
      await apiDelete(`/novels/${novelId}`);
      setToast('小说已删除。');
    } catch (error) {
      setToast(getErrorMessage(error, '删除接口暂不可用，已先从本地列表移除。'));
    }
  };

  return (
    <div className="app-shell">
      <div className="phone-frame">
        {tab === 'creation' ? (
          <main className="page">{screen === 'list' && <NovelList novels={novels} onOpen={openWorkflow} onCreate={createNovel} onDelete={deleteNovel} toast={toast} />}{screen === 'workflow' && selectedNovel && <WorkflowView novel={selectedNovel} onBack={() => setScreen('list')} onOpenStep={openStep} toast={toast} />}{screen === 'step' && selectedNovel && selectedStep && <StepView novel={selectedNovel} step={selectedStep} status={getStepStatus(selectedNovel, steps.findIndex((item) => item.key === selectedStep.key))} suggestion={suggestion} content={stepContent} loading={loading} onSuggestion={setSuggestion} onVoice={useVoiceInput} onSend={sendSuggestion} onSaveDraft={saveDraft} onBack={() => setScreen('workflow')} onArchive={() => setArchiveDialog(true)} onEditArchived={() => setEditDialog(true)} toast={toast} />}</main>
        ) : (
          <ProfileView profile={profile} stats={profileStats} toast={toast} />
        )}
        <nav className="bottom-nav" aria-label="底部导航">
          <button className={tab === 'creation' ? 'nav-active' : ''} onClick={() => setTab('creation')}>创作中心</button>
          <button className={tab === 'profile' ? 'nav-active' : ''} onClick={() => setTab('profile')}>个人</button>
        </nav>
      </div>
      {archiveDialog && selectedStep && <ArchiveDialog step={selectedStep} onCancel={() => setArchiveDialog(false)} onConfirm={confirmArchive} />}
      {editDialog && selectedStep && <EditWarningDialog step={selectedStep} onCancel={() => setEditDialog(false)} onConfirm={confirmEditArchived} />}
    </div>
  );
}

function NovelList({ novels, onOpen, onCreate, onDelete, toast }: { novels: Novel[]; onOpen: (id: number) => void; onCreate: () => void; onDelete: (id: number) => void; toast: string }) {
  return (
    <section className="content-pad">
      <header className="page-header">
        <div>
          <h1>创作中心</h1>
          <p>选择一部小说进入创作流程</p>
        </div>
        <button className="icon-button" aria-label="创建小说" onClick={onCreate}><Plus size={18} /></button>
      </header>
      {toast && <div className="toast">{toast}</div>}
      <div className="card-list">
        {novels.map((novel) => {
          const currentStep = steps[Math.min(novel.archivedCount, steps.length - 1)];
          return (
            <article className="novel-card" key={novel.id}>
              <div>
                <h2>{novel.title}</h2>
                <p>{novel.genre} · 当前步骤：{currentStep.name}</p>
                <p>进度：{novel.archivedCount}/6 已归档 · {novel.updatedAt}</p>
              </div>
              <div className="card-actions">
                <button className="primary-button" onClick={() => onOpen(novel.id)}>进入流程</button>
                <button className="text-button" onClick={() => onDelete(novel.id)}>删除</button>
              </div>
            </article>
          );
        })}
      </div>
    </section>
  );
}

function WorkflowView({ novel, onBack, onOpenStep, toast }: { novel: Novel; onBack: () => void; onOpenStep: (step: StepConfig) => void; toast: string }) {
  return (
    <section className="content-pad">
      <HeaderBack title={novel.title} subtitle={`${novel.archivedCount}/6 已归档 · 小说创作流程`} onBack={onBack} />
      <h2 className="section-title">小说创作流程</h2>
      {toast && <div className="toast">{toast}</div>}
      <div className="step-list">
        {steps.map((step, index) => {
          const status = getStepStatus(novel, index);
          return (
            <button aria-label={step.name} className={`step-card status-${status}`} key={step.key} onClick={() => onOpenStep(step)}>
              <span className="step-mark">{status === 'archived' ? '✓' : status === 'active' ? '●' : status === 'review' ? '!' : '○'}</span>
              <span>
                <strong>{step.name}</strong>
                <small>{status === 'archived' ? '已归档 · 可查看/申请修改' : status === 'active' ? '当前步骤 · 继续创作' : status === 'review' ? '需复核 · 上游内容已修改' : `请先归档${steps[index - 1]?.name}`}</small>
              </span>
            </button>
          );
        })}
      </div>
    </section>
  );
}

function StepView({ novel, step, status, suggestion, content, loading, onSuggestion, onVoice, onSend, onSaveDraft, onBack, onArchive, onEditArchived, toast }: { novel: Novel; step: StepConfig; status: StepStatus; suggestion: string; content: StepContent | null; loading: boolean; onSuggestion: (value: string) => void; onVoice: () => void; onSend: () => void; onSaveDraft: () => void; onBack: () => void; onArchive: () => void; onEditArchived: () => void; toast: string }) {
  const visibleContent = content ?? { lines: step.sampleContent, upstreamSummary: '这里展示当前步骤所需的上游归档摘要，确保创作不跳过依赖。', aiTip: step.aiTip };
  return (
    <section className="step-page">
      <div className="content-pad step-scroll">
        <HeaderBack title={step.name} subtitle={`${novel.title} · ${step.upstreamTitle}`} onBack={onBack} />
        {toast && <div className="risk-bar">{toast}</div>}
        <div className="mini-card muted-card">
          <span>{step.upstreamTitle}</span>
          <p>{visibleContent.upstreamSummary}</p>
        </div>
        <section className="block-card">
          <div className="block-title"><Archive size={16} /> 创作内容</div>
          {loading ? <p>正在读取步骤内容...</p> : visibleContent.lines.map((line) => <p key={line}>{line}</p>)}
        </section>
        <section className="block-card">
          <div className="block-title">AI 协作</div>
          <p>{visibleContent.aiTip}</p>
        </section>
      </div>
      <div className="composer">
        <div className="input-row">
          <button aria-label="语音输入" className="mic-button" onClick={onVoice}><Mic size={18} /></button>
          <input value={suggestion} placeholder={step.inputPlaceholder} onChange={(event) => onSuggestion(event.target.value)} />
          <button aria-label="发送建议" className="send-button" onClick={onSend}><Send size={16} /></button>
        </div>
        <div className="composer-actions">
          <button className="secondary-button" onClick={onSaveDraft}>保存草稿</button>
          {status === 'archived' ? <button className="primary-button" onClick={onEditArchived}>修改归档</button> : <button className="primary-button" onClick={onArchive}>{step.archiveText}</button>}
        </div>
      </div>
    </section>
  );
}

function ProfileView({ profile, stats, toast }: { profile: Profile | null; stats: ProfileStats | null; toast: string }) {
  const visibleProfile = profile ?? { id: 0, nickname: '创作者昵称', avatar_url: '', bio: '这里将展示用户简介' };
  const visibleStats = stats ?? { novel_count: 0, word_count: 0 };
  return (
    <main className="page content-pad">
      <header className="page-header profile-header">
        <div>
          <h1>个人中心</h1>
          <p>管理你的创作身份</p>
        </div>
      </header>
      {toast && <div className="toast">{toast}</div>}
      <section className="profile-card">
        <div className="avatar"><UserRound size={24} /></div>
        <h2>{visibleProfile.nickname}</h2>
        <p>ID：{visibleProfile.id || '--'}</p>
        <p>{visibleProfile.bio || '这个作者暂时还没有简介'}</p>
      </section>
      <section className="block-card"><div className="block-title">创作数据</div><p>作品数：{visibleStats.novel_count}</p><p>累计字数：{visibleStats.word_count}</p></section>
      <section className="block-card"><div className="block-title">更多能力</div><p>设置、会员、反馈等入口将在后续版本开放。</p></section>
    </main>
  );
}

function HeaderBack({ title, subtitle, onBack }: { title: string; subtitle: string; onBack: () => void }) {
  return (
    <header className="detail-header">
      <button className="back-button" onClick={onBack} aria-label="返回"><ChevronLeft size={20} /></button>
      <div><h1>{title}</h1><p>{subtitle}</p></div>
    </header>
  );
}

function ArchiveDialog({ step, onCancel, onConfirm }: { step: StepConfig; onCancel: () => void; onConfirm: () => void }) {
  const next = steps[steps.findIndex((item) => item.key === step.key) + 1];
  return <Dialog title={`归档${step.shortName}？`} body={`归档后将解锁下一步“${next?.name ?? '作品完成'}”。后续创作会基于当前归档内容。`} confirm="确认归档" onCancel={onCancel} onConfirm={onConfirm} />;
}

function EditWarningDialog({ step, onCancel, onConfirm }: { step: StepConfig; onCancel: () => void; onConfirm: () => void }) {
  return <Dialog title="修改已归档内容？" body={`你正在修改“${step.name}”的归档版。修改归档内容会影响下面步骤已经创作的内容，请确认你已经理解风险。`} confirm="我理解，继续修改" onCancel={onCancel} onConfirm={onConfirm} />;
}

function Dialog({ title, body, confirm, onCancel, onConfirm }: { title: string; body: string; confirm: string; onCancel: () => void; onConfirm: () => void }) {
  return (
    <div className="dialog-mask" role="presentation">
      <section className="dialog" role="dialog" aria-modal="true" aria-label={title}>
        <h2>{title}</h2>
        <p>{body}</p>
        <div className="dialog-actions">
          <button className="secondary-button" onClick={onCancel}>取消</button>
          <button className="primary-button" onClick={onConfirm}>{confirm}</button>
        </div>
      </section>
    </div>
  );
}

async function apiGet<T>(path: string): Promise<T> {
  return apiRequest<T>(path, { method: 'GET' });
}

async function apiPost<T = unknown>(path: string, body: unknown): Promise<T> {
  return apiRequest<T>(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
}

async function apiPut<T = unknown>(path: string, body: unknown): Promise<T> {
  return apiRequest<T>(path, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
}

async function apiDelete<T = unknown>(path: string): Promise<T> {
  return apiRequest<T>(path, { method: 'DELETE' });
}

async function apiRequest<T>(path: string, init: RequestInit): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, init);
  if (!response.ok) {
    // 后端统一返回中文 message；解析失败时回退状态码，保证面向前端的错误提示始终可读。
    const payload = await safeJson<{ message?: string }>(response);
    throw new Error(payload?.message || `接口请求失败：${response.status}`);
  }
  // DELETE 等接口当前也返回 JSON；这里保留空响应兼容，便于后续接入无正文的下游接口。
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

async function safeJson<T>(response: Response): Promise<T | null> {
  try {
    return (await response.clone().json()) as T;
  } catch {
    return null;
  }
}

function getErrorMessage(error: unknown, fallback: string): string {
  return error instanceof Error && error.message ? error.message : fallback;
}

function normalizeNovel(novel: NovelApiItem): Novel {
  return {
    id: novel.id,
    title: novel.title,
    genre: novel.genre || '未分类',
    updatedAt: formatDate(novel.updated_at),
    archivedCount: novel.archived_count ?? 0,
  };
}

function normalizeStepContent(step: StepConfig, payload: Record<string, unknown>): StepContent {
  const draft = payload.draft as Record<string, unknown> | undefined;
  const upstream = payload.upstream_summary ? JSON.stringify(payload.upstream_summary) : '已读取上游归档摘要。';
  const lines = collectReadableLines(draft);
  return {
    lines: lines.length > 0 ? lines : step.sampleContent,
    upstreamSummary: upstream,
    aiTip: `已从接口读取“${step.name}”内容，可继续发送建议或保存草稿。`,
  };
}

function collectReadableLines(value: unknown): string[] {
  if (!value) return [];
  if (typeof value === 'string') return value.trim() ? [value] : [];
  if (Array.isArray(value)) return value.flatMap(collectReadableLines).slice(0, 8);
  if (typeof value !== 'object') return [];
  const record = value as Record<string, unknown>;
  const preferredKeys = ['title', 'synopsis', 'outline', 'theme', 'description', 'content', 'name'];
  const direct = preferredKeys.flatMap((key) => (typeof record[key] === 'string' && record[key] ? [`${labelForKey(key)}：${record[key]}`] : []));
  const nested = Object.entries(record)
    .filter(([, nestedValue]) => Array.isArray(nestedValue))
    .flatMap(([key, nestedValue]) => collectReadableLines(nestedValue).map((line) => `${labelForKey(key)}｜${line}`));
  return [...direct, ...nested].slice(0, 10);
}

function labelForKey(key: string) {
  const labels: Record<string, string> = { title: '标题', synopsis: '简介', outline: '大纲', theme: '主题', description: '描述', content: '正文', name: '名称', plotlines: '剧情线', characters: '人物', volumes: '分卷', chapters: '章节' };
  return labels[key] ?? key;
}

function formatDate(value: string) {
  if (!value) return '刚刚更新';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' });
}
