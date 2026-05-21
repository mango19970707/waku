import { useMemo, useState } from 'react';
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

  const selectedNovel = useMemo(
    () => novels.find((novel) => novel.id === selectedNovelId) ?? null,
    [novels, selectedNovelId],
  );
  const selectedStep = steps.find((step) => step.key === selectedStepKey) ?? null;

  const openWorkflow = (novelId: number) => {
    setSelectedNovelId(novelId);
    setScreen('workflow');
    setToast('');
  };

  const openStep = (step: StepConfig) => {
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
  };

  const confirmArchive = () => {
    if (!selectedNovel || !selectedStep) return;
    const index = steps.findIndex((step) => step.key === selectedStep.key);
    setNovels((current) =>
      current.map((novel) =>
        novel.id === selectedNovel.id
          ? { ...novel, archivedCount: Math.max(novel.archivedCount, index + 1), reviewFrom: undefined }
          : novel,
      ),
    );
    setArchiveDialog(false);
    setScreen('workflow');
  };

  const confirmEditArchived = () => {
    if (!selectedNovel || !selectedStep) return;
    setNovels((current) =>
      current.map((novel) => (novel.id === selectedNovel.id ? { ...novel, reviewFrom: selectedStep.key } : novel)),
    );
    setEditDialog(false);
    setToast('正在修改归档内容，下游步骤可能需要重新检查。');
  };

  const useVoiceInput = () => {
    setSuggestion('帮我增加一个亦敌亦友角色');
  };

  return (
    <div className="app-shell">
      <div className="phone-frame">
        {tab === 'creation' ? (
          <main className="page">{screen === 'list' && <NovelList novels={novels} onOpen={openWorkflow} />}{screen === 'workflow' && selectedNovel && <WorkflowView novel={selectedNovel} onBack={() => setScreen('list')} onOpenStep={openStep} toast={toast} />}{screen === 'step' && selectedNovel && selectedStep && <StepView novel={selectedNovel} step={selectedStep} status={getStepStatus(selectedNovel, steps.findIndex((item) => item.key === selectedStep.key))} suggestion={suggestion} onSuggestion={setSuggestion} onVoice={useVoiceInput} onBack={() => setScreen('workflow')} onArchive={() => setArchiveDialog(true)} onEditArchived={() => setEditDialog(true)} toast={toast} />}</main>
        ) : (
          <ProfileView />
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

function NovelList({ novels, onOpen }: { novels: Novel[]; onOpen: (id: number) => void }) {
  return (
    <section className="content-pad">
      <header className="page-header">
        <div>
          <h1>创作中心</h1>
          <p>选择一部小说进入创作流程</p>
        </div>
        <button className="icon-button" aria-label="创建小说"><Plus size={18} /></button>
      </header>
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
                <button className="text-button">删除</button>
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

function StepView({ novel, step, status, suggestion, onSuggestion, onVoice, onBack, onArchive, onEditArchived, toast }: { novel: Novel; step: StepConfig; status: StepStatus; suggestion: string; onSuggestion: (value: string) => void; onVoice: () => void; onBack: () => void; onArchive: () => void; onEditArchived: () => void; toast: string }) {
  return (
    <section className="step-page">
      <div className="content-pad step-scroll">
        <HeaderBack title={step.name} subtitle={`${novel.title} · ${step.upstreamTitle}`} onBack={onBack} />
        {toast && <div className="risk-bar">{toast}</div>}
        <div className="mini-card muted-card">
          <span>{step.upstreamTitle}</span>
          <p>这里展示当前步骤所需的上游归档摘要，确保创作不跳过依赖。</p>
        </div>
        <section className="block-card">
          <div className="block-title"><Archive size={16} /> 创作内容</div>
          {step.sampleContent.map((line) => <p key={line}>{line}</p>)}
        </section>
        <section className="block-card">
          <div className="block-title">AI 协作</div>
          <p>{step.aiTip}</p>
        </section>
      </div>
      <div className="composer">
        <div className="input-row">
          <button aria-label="语音输入" className="mic-button" onClick={onVoice}><Mic size={18} /></button>
          <input value={suggestion} placeholder={step.inputPlaceholder} onChange={(event) => onSuggestion(event.target.value)} />
          <button aria-label="发送建议" className="send-button"><Send size={16} /></button>
        </div>
        <div className="composer-actions">
          <button className="secondary-button">保存草稿</button>
          {status === 'archived' ? <button className="primary-button" onClick={onEditArchived}>修改归档</button> : <button className="primary-button" onClick={onArchive}>{step.archiveText}</button>}
        </div>
      </div>
    </section>
  );
}

function ProfileView() {
  return (
    <main className="page content-pad">
      <header className="page-header profile-header">
        <div>
          <h1>个人中心</h1>
          <p>管理你的创作身份</p>
        </div>
      </header>
      <section className="profile-card">
        <div className="avatar"><UserRound size={24} /></div>
        <h2>创作者昵称</h2>
        <p>ID：--</p>
        <p>这里将展示用户简介</p>
      </section>
      <section className="block-card"><div className="block-title">创作数据</div><p>作品数：--</p><p>累计字数：--</p></section>
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
