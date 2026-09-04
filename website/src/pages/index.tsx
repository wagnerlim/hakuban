import type {ReactNode} from 'react';
import Layout from '@theme/Layout';
import {translate} from '@docusaurus/Translate';
import useBaseUrl from '@docusaurus/useBaseUrl';
import styles from './index.module.css';

/**
 * Homepage. Every visible string goes through translate() with a literal id so
 * `docusaurus write-translations` can extract it. Hard-coded on purpose (identical in all
 * three locales): Hakuban, 白板, the field names, the column labels, the install command
 * and the two comparison-column headers.
 */
function useStrings() {
  return {
    eyebrow: translate({id: 'home.eyebrow', message: 'hakuban, “white board”'}),
    heroLead: translate({
      id: 'home.heroLead',
      message:
        'A kanban in your terminal that your agents work on too — and where you declare what happens when a card changes column.',
    }),
    heroSub: translate({
      id: 'home.heroSub',
      message:
        'Every card and every board is a Markdown file on disk, so an AI reads and edits all of it the same way you do — it is just text. Columns are states, and each state is yours to define: run a script, send an email, tag a release, hand the task off. Hakuban ships the board; the flow is a blank page.',
    }),
    cta: translate({id: 'home.cta', message: 'See the board format →'}),
    expectation: translate({
      id: 'home.expectation',
      message: 'Solo project · no support · PRs may sit · MIT',
    }),
    assetTag: translate({id: 'home.assetTag', message: 'drop asset here'}),
    assetTitle: translate({
      id: 'home.assetTitle',
      message: 'asciinema — a card crossing a column',
    }),
    assetNote: translate({
      id: 'home.assetNote',
      message:
        'The TUI live while an agent moves cards, and a hand-drag doing the same thing. The point is the transition, not the payload.',
    }),

    boardKicker: translate({id: 'home.boardKicker', message: 'The board'}),
    boardTitle: translate({
      id: 'home.boardTitle',
      message: 'A state machine that lives on disk.',
    }),
    boardLead: translate({
      id: 'home.boardLead',
      message:
        'The board is one .md with YAML frontmatter — the columns and what each one does — and every card is another .md: frontmatter plus the markdown body where the notes live. The disk is the source of truth, and it has two legitimate writers — the TUI, and whoever edits the file by hand: you, or an agent. Writes are atomic; the TUI re-reads the disk about once a second, so a card an agent moves appears in front of you.',
    }),
    yamlGuide: translate({id: 'home.yamlGuide', message: 'One card here at a time.'}),
    yamlInstruction: translate({
      id: 'home.yamlInstruction',
      message: 'Whatever should happen, in your own words.',
    }),
    fieldColumns: translate({
      id: 'home.fieldColumns',
      message: 'The states of the belt, in order.',
    }),
    fieldOnEnter: translate({
      id: 'home.fieldOnEnter',
      message: 'Instruction — prose run by a headless agent.',
    }),
    fieldCmd: translate({
      id: 'home.fieldCmd',
      message: 'Script — shell, card as JSON on stdin.',
    }),
    fieldTools: translate({
      id: 'home.fieldTools',
      message: 'The only tools an instruction on this board may use.',
    }),
    fieldGuide: translate({
      id: 'home.fieldGuide',
      message: 'A note for the column. Read, never executed.',
    }),
    contractTag: translate({id: 'home.contractTag', message: 'the contract'}),
    contractBody: translate({
      id: 'home.contractBody',
      message:
        'A hook that fails aborts the move. The card stays where it was and the reason shows up — same whether a human dragged it or an agent called hakuban move.',
    }),
    cardAgentsTitle: translate({id: 'home.cardAgentsTitle', message: 'agents, by default'}),
    cardAgentsBody: translate({
      id: 'home.cardAgentsBody',
      message:
        'The headless subcommands move and progress let an agent drive the belt with no terminal attached.',
    }),
    cardHandsTitle: translate({id: 'home.cardHandsTitle', message: 'hands, when you want'}),
    cardHandsBody: translate({
      id: 'home.cardHandsBody',
      message:
        'Drag a card and the same hooks fire. Draggable shows grab, dragging shows grabbing; one click selects, two activate.',
    }),
    cardHttpTitle: translate({
      id: 'home.cardHttpTitle',
      message: 'the binary speaks no HTTP',
    }),
    cardHttpBody: translate({
      id: 'home.cardHttpBody',
      message:
        'Hakuban never touches the network. A hook does. Integration is composition — a principle, not a gap.',
    }),

    exKicker: translate({id: 'home.exKicker', message: 'What to declare'}),
    exTitle: translate({
      id: 'home.exTitle',
      message: 'Four boards. None ship with the tool.',
    }),
    exLead: translate({
      id: 'home.exLead',
      message:
        "Examples, not features. The author's own board hands tasks to an agent that opens PRs — one flow among N, and yours has no reason to look like it.",
    }),
    ex1Title: translate({id: 'home.ex1Title', message: 'Email the client'}),
    ex1Body: translate({
      id: 'home.ex1Body',
      message:
        '“Write a short update from the card notes and send it to the address in the frontmatter.”',
    }),
    ex2Title: translate({id: 'home.ex2Title', message: 'Split the card'}),
    ex2Body: translate({
      id: 'home.ex2Body',
      message:
        '“If the notes describe more than one deliverable, create one card per deliverable and leave this one as the parent.”',
    }),
    ex3Title: translate({id: 'home.ex3Title', message: 'Run the checklist'}),
    ex3Body: translate({
      id: 'home.ex3Body',
      message:
        'A shell script: tests, linter, coverage floor. Non-zero exit and the card never leaves Doing.',
    }),
    ex4Title: translate({id: 'home.ex4Title', message: 'Hand it to an agent'}),
    ex4Body: translate({
      id: 'home.ex4Body',
      message:
        "The author's board went card → implementation → approved PR in 9m18s on 2026-08-29. Real, and still just an example.",
    }),
    honestTag: translate({id: 'home.honestTag', message: 'honest'}),
    honestBody: translate({
      id: 'home.honestBody',
      message:
        'On a clean machine the agent demo does not run: it needs jq, an authenticated gh, a paid Claude Code subscription, and the herdr multiplexer. The pitch is “see how it’s done”, not “install and use”.',
    }),

    distKicker: translate({id: 'home.distKicker', message: 'The distinction'}),
    distTitle: translate({id: 'home.distTitle', message: 'Instruction is not script.'}),
    distLead: translate({
      id: 'home.distLead',
      message:
        'Both slots fire on the same transition, so they look interchangeable. They are not. For sharing a board, what matters is not what the file says — it is what it can do.',
    }),
    colScript: translate({id: 'home.colScript', message: 'deterministic shell'}),
    colInstruction: translate({
      id: 'home.colInstruction',
      message: 'prose, headless agent',
    }),
    rowBehavior: translate({id: 'home.rowBehavior', message: 'behavior'}),
    behaviorScript: translate({
      id: 'home.behaviorScript',
      message: 'Deterministic — you read exactly what it does.',
    }),
    behaviorInstruction: translate({
      id: 'home.behaviorInstruction',
      message: "Non-deterministic — you don't know what it will do.",
    }),
    rowCapability: translate({id: 'home.rowCapability', message: 'capability'}),
    capScript: translate({
      id: 'home.capScript',
      message: 'Unlimited — there is no permission model in bash.',
    }),
    capInstruction: translate({
      id: 'home.capInstruction',
      message: 'Bounded — it fits on one line: agent_tools.',
    }),
    rowReview: translate({id: 'home.rowReview', message: 'review scales?'}),
    reviewScript: translate({
      id: 'home.reviewScript',
      message: 'No — nobody reviews 93 invocations of jq.',
    }),
    reviewInstruction: translate({
      id: 'home.reviewInstruction',
      message: 'Yes — a capability ceiling is a short list.',
    }),
    rowPortability: translate({id: 'home.rowPortability', message: 'portability'}),
    portScript: translate({
      id: 'home.portScript',
      message: 'Ties you to a filesystem, a binary, a $HOME.',
    }),
    portInstruction: translate({
      id: 'home.portInstruction',
      message: 'Ties you to nothing: it describes intent.',
    }),
    doctrineTag: translate({id: 'home.doctrineTag', message: 'Project doctrine:'}),
    doctrineBody: translate({
      id: 'home.doctrineBody',
      message:
        'the instruction is the format; _cmd is debt — an escape hatch for exact control, at the cost of a board nobody else can run.',
    }),
  };
}

type Strings = ReturnType<typeof useStrings>;

/**
 * The recording in the hero slot. `null` keeps the dashed placeholder.
 *
 * The author's own demo. It does NOT autoplay: it runs a minute and weighs 11 MB, so the
 * landing page would pay for a download most visitors never watch. `preload="metadata"`
 * plus a real frame as the poster means the hero shows a live board immediately and the
 * file only downloads on play.
 *
 * The video itself is not in git — it comes from a release, see website/fetch-assets.sh.
 * The poster is, so the hero still shows the board if that fetch ever fails.
 */
const ASSET: {src: string; type: 'video' | 'image'; poster?: string} | null = {
  src: '/demo.mp4',
  type: 'video',
  poster: '/demo-poster.jpg',
};

function AssetFigure({s}: {s: Strings}) {
  // The site is served under a baseUrl, so a root-relative asset path has to be resolved.
  const src = useBaseUrl(ASSET?.src ?? '');
  const poster = useBaseUrl(ASSET?.poster ?? '');
  return (
    <figure className={styles.figure}>
      <div className={styles.titlebar}>
        <span className={styles.dot} style={{background: '#ff5f57'}} />
        <span className={styles.dot} style={{background: '#febc2e'}} />
        <span className={styles.dot} style={{background: '#28c840'}} />
        <span className={styles.titlebarLabel}>hakuban — project-board</span>
      </div>
      {ASSET ? (
        <div className={styles.figureBodyFilled}>
          {ASSET.type === 'video' ? (
            <video
              className={styles.asset}
              src={src}
              poster={ASSET.poster ? poster : undefined}
              controls
              preload="metadata"
              playsInline
            />
          ) : (
            <img className={styles.asset} src={src} alt={s.assetTitle} />
          )}
        </div>
      ) : (
        <div className={styles.figureBody}>
          <div className={styles.placeholder}>
            <div className={styles.placeholderTag}>{s.assetTag}</div>
            <div className={styles.placeholderTitle}>{s.assetTitle}</div>
            <div className={styles.placeholderNote}>{s.assetNote}</div>
          </div>
        </div>
      )}
    </figure>
  );
}

function Hero({s}: {s: Strings}) {
  return (
    <section className={styles.hero}>
      <div className={styles.heroStack}>
        <div className={styles.eyebrow}>白板 — {s.eyebrow}</div>
        <div className={styles.logoRow}>
          <span className={styles.logoKanji}>
            <span className="lockup-solid">白</span>
            <span className="lockup-outline">板</span>
          </span>
          <h1 className={styles.h1}>Hakuban</h1>
        </div>
        <p className={styles.lead}>{s.heroLead}</p>
        <p className={styles.sub}>{s.heroSub}</p>
        <div className={styles.ctaRow}>
          <a className={styles.cta} href="#board">
            {s.cta}
          </a>
          <code className={styles.installChip}>
            <span className={styles.prompt}>$</span>go install
            github.com/wagnerlim/hakuban/cmd/hakuban@latest
          </code>
        </div>
        <div className={styles.expectation}>{s.expectation}</div>
      </div>
      <AssetFigure s={s} />
    </section>
  );
}

function BoardYaml({s}: {s: Strings}) {
  return (
    <pre className={styles.yaml}>
      <span className={styles.yDim}># ~/.hakuban/boards/my-board.md{'\n'}</span>
      <span className={styles.yMuted}>---{'\n'}</span>
      <span className={styles.yKey}>name</span>: My board{'\n'}
      <span className={styles.yKey}>columns</span>: [To-Do, Doing, Review, Done]{'\n'}
      <span className={styles.yKey}>actions</span>:{'\n'}
      {'  '}
      <span className={styles.yKey}>Doing</span>:{'\n'}
      {'    '}
      <span className={styles.yMuted}>guide</span>: {s.yamlGuide}
      {'\n'}
      {'    '}
      <span className={styles.yMuted}>on_enter</span>: |{'\n'}
      {'      '}
      {s.yamlInstruction}
      {'\n'}
      {'  '}
      <span className={styles.yKey}>Done</span>:{'\n'}
      {'    '}
      <span className={styles.yMuted}>on_enter_cmd</span>: hooks/notify.sh{'\n'}
      <span className={styles.yKey}>agent_tools</span>: Read, Write, Bash{'\n'}
      <span className={styles.yMuted}>---</span>
    </pre>
  );
}

function Board({s}: {s: Strings}) {
  const fields: [string, string][] = [
    ['columns', s.fieldColumns],
    ['on_enter / on_exit', s.fieldOnEnter],
    ['*_cmd', s.fieldCmd],
    ['agent_tools', s.fieldTools],
    ['guide', s.fieldGuide],
  ];
  const cards: [string, string][] = [
    [s.cardAgentsTitle, s.cardAgentsBody],
    [s.cardHandsTitle, s.cardHandsBody],
    [s.cardHttpTitle, s.cardHttpBody],
  ];

  return (
    <section id="board" className={styles.section}>
      <div className={styles.sectionHead}>
        <div className={styles.kicker}>{s.boardKicker}</div>
        <h2 className={styles.h2}>{s.boardTitle}</h2>
        <p className={styles.sectionLead}>{s.boardLead}</p>
      </div>

      <div className={styles.boardGrid}>
        <BoardYaml s={s} />
        <div className={styles.fieldBox}>
          {fields.map(([name, desc]) => (
            <div className={styles.fieldRow} key={name}>
              <code className={styles.fieldName}>{name}</code>
              <span className={styles.fieldDesc}>{desc}</span>
            </div>
          ))}
          <div className={styles.contract}>
            <div className={styles.contractTag}>{s.contractTag}</div>
            <span className={styles.contractBody}>{s.contractBody}</span>
          </div>
        </div>
      </div>

      <div className={styles.cards3}>
        {cards.map(([title, body]) => (
          <div className={styles.card} key={title}>
            <div className={styles.cardTitleMono}>{title}</div>
            <div className={styles.cardBody}>{body}</div>
          </div>
        ))}
      </div>
    </section>
  );
}

function Examples({s}: {s: Strings}) {
  const examples: [string, string, string][] = [
    ['Done →', s.ex1Title, s.ex1Body],
    ['Refining →', s.ex2Title, s.ex2Body],
    ['Review →', s.ex3Title, s.ex3Body],
    ['Doing →', s.ex4Title, s.ex4Body],
  ];

  return (
    <section id="examples" className={styles.band}>
      <div className={styles.bandInner}>
        <div className={styles.sectionHead}>
          <div className={styles.kicker}>{s.exKicker}</div>
          <h2 className={styles.h2}>{s.exTitle}</h2>
          <p className={styles.sectionLead}>{s.exLead}</p>
        </div>

        <div className={styles.cards4}>
          {examples.map(([label, title, body], i) => {
            const last = i === examples.length - 1;
            return (
              <div className={last ? styles.exCardEmphasis : styles.exCard} key={title}>
                <div className={last ? styles.exLabelEmphasis : styles.exLabel}>{label}</div>
                <div className={styles.exTitle}>{title}</div>
                <div className={styles.exBody}>{body}</div>
              </div>
            );
          })}
        </div>

        <div className={styles.honest}>
          <span className={styles.honestTag}>{s.honestTag}</span>
          <span className={styles.honestBody}>{s.honestBody}</span>
        </div>
      </div>
    </section>
  );
}

const SCRIPT_COL = 'script · _cmd';
const INSTRUCTION_COL = 'instruction · on_enter';

function Distinction({s}: {s: Strings}) {
  const rows: [string, string, string][] = [
    [s.rowBehavior, s.behaviorScript, s.behaviorInstruction],
    [s.rowCapability, s.capScript, s.capInstruction],
    [s.rowReview, s.reviewScript, s.reviewInstruction],
    [s.rowPortability, s.portScript, s.portInstruction],
  ];

  return (
    <section id="instruction" className={styles.section}>
      <div className={styles.sectionHead}>
        <div className={styles.kicker}>{s.distKicker}</div>
        <h2 className={styles.h2}>{s.distTitle}</h2>
        <p className={styles.sectionLead}>{s.distLead}</p>
      </div>

      <div className={styles.table}>
        <div className={styles.thCorner} />
        <div className={styles.thScript}>
          <div className={styles.thName}>{SCRIPT_COL}</div>
          <div className={styles.thSub}>{s.colScript}</div>
        </div>
        <div className={styles.thInstruction}>
          <div className={styles.thName}>{INSTRUCTION_COL}</div>
          <div className={styles.thSub}>{s.colInstruction}</div>
        </div>

        {rows.map(([label, script, instruction], i) => {
          const last = i === rows.length - 1;
          return (
            <div className={styles.tr} key={label}>
              <div className={last ? styles.tdLabelLast : styles.tdLabel}>{label}</div>
              <div className={last ? styles.tdLast : styles.td} data-col={SCRIPT_COL}>
                {script}
              </div>
              <div className={last ? styles.tdLast : styles.td} data-col={INSTRUCTION_COL}>
                {instruction}
              </div>
            </div>
          );
        })}
      </div>

      <p className={styles.doctrine}>
        <span className={styles.doctrineTag}>{s.doctrineTag}</span> {s.doctrineBody}
      </p>
    </section>
  );
}

export default function Home(): ReactNode {
  const s = useStrings();
  return (
    <Layout title="Hakuban" description={s.heroLead}>
      <Hero s={s} />
      <Board s={s} />
      <Examples s={s} />
      <Distinction s={s} />
    </Layout>
  );
}
