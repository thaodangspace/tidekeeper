
<script lang="ts">
  const features = [
    {
      icon: '✦',
      eyebrow: 'READ THE TIDE',
      title: 'Every day brings a new current',
      text: 'Real-world market movement is transformed into fantasy signals, modifiers, and objectives—never direct trading.'
    },
    {
      icon: '⚓',
      eyebrow: 'BUILD YOUR FLEET',
      title: 'Choose Keepers, roles, and synergies',
      text: 'Combine Wardens, Oracles, Vanguards, and Supports to create a fleet that can survive more than one kind of storm.'
    },
    {
      icon: '⌛',
      eyebrow: 'LOCK & RETURN',
      title: 'A strategy game that respects your time',
      text: 'Prepare in minutes, lock your fleet, then return the next day to see what survived—and why.'
    }
  ];

  const steps = [
    ['01', 'Read today’s signals', 'Study the current, objective, and daily modifier before the lock closes.'],
    ['02', 'Shape your fleet', 'Recruit Keepers, activate synergies, and choose a one-day strategy.'],
    ['03', 'Lock your formation', 'Your lineup becomes an immutable snapshot for the coming Tide.'],
    ['04', 'Face tomorrow', 'Return for the result, breakdown, rewards, and the next step of your Voyage.']
  ];

  const keepers = [
    { name: 'Crest Guardian', role: 'Vanguard', tone: 'crest', image: '/art/crest-guardian.webp', text: 'Anchors the fleet when the World Tide turns against you.' },
    { name: 'Harbor Warden', role: 'Warden', tone: 'harbor', image: '/art/harbor-warden.webp', text: 'Controls Depth and protects the fleet’s most exposed Keeper.' },
    { name: 'Echo Seer', role: 'Oracle', tone: 'echo', image: '/art/echo-seer.webp', text: 'Reveals more information without promising the future.' },
    { name: 'Current Weaver', role: 'Support', tone: 'current', image: '/art/current-weaver.webp', text: 'Rewards balanced fleets and transfers momentum across sectors.' }
  ];
</script>

<svelte:head>
  <title>Tidekeepers — Command the Tide. Defy the Deep.</title>
  <meta name="description" content="Tidekeepers is a daily asynchronous strategy roguelite where you build a fleet, read the tides, and return tomorrow to discover the outcome." />
</svelte:head>

<div class="page-shell">
  <header class="site-header">
    <a class="brand" href="#top" aria-label="Tidekeepers home">
      <span class="brand-mark">✦</span>
      <span class="brand-word">TIDEKEEPERS</span>
    </a>
    <nav aria-label="Primary navigation">
      <a href="#gameplay">Gameplay</a>
      <a href="#keepers">Keepers</a>
      <a href="#voyage">Voyage</a>
    </nav>
    <a class="button button-small button-outline" href="#join">Enter the Voyage</a>
  </header>

  <main id="top">
    <section class="hero section-frame">
      <div class="hero-copy">
        <p class="kicker">DAILY ASYNCHRONOUS STRATEGY ROGUELITE</p>
        <h1>Command the Tide.<br /><span>Defy the Deep.</span></h1>
        <p class="hero-lead">
          Read the Tide. Assemble a fleet of Keepers. Lock your strategy before dawn—then return tomorrow to discover whether your line held.
        </p>
        <div class="hero-actions">
          <a class="button button-primary" href="#join">Begin Your Voyage</a>
          <a class="text-link" href="#gameplay">See how it plays <span>→</span></a>
        </div>
        <div class="hero-stats" aria-label="Game highlights">
          <div><strong>3–5 min</strong><span>daily session</span></div>
          <div><strong>7 days</strong><span>one Voyage</span></div>
          <div><strong>No trading</strong><span>pure strategy</span></div>
        </div>
      </div>

      <div class="hero-visual" aria-label="Tidekeepers game dashboard preview">
        <div class="visual-glow"></div>
        <div class="dashboard-frame">
          <img src="/tidekeepers-dashboard.png" alt="Tidekeepers fleet preparation dashboard" />
        </div>
        <div class="floating-card floating-card-left">
          <span class="mini-label">TODAY'S TIDE</span>
          <strong>Flight to Harbor</strong>
          <small>Wardens gain defense</small>
        </div>
        <div class="floating-card floating-card-right">
          <span class="mini-label">VOYAGE</span>
          <strong>Day 3 / 7</strong>
          <small>Hull integrity: 82%</small>
        </div>
      </div>
    </section>

    <section class="trust-strip" aria-label="Product positioning">
      <span>REAL-WORLD DATA, FANTASY RULES</span>
      <i></i>
      <span>NO REAL-MONEY TRADING, NO WEB3, NO NFT</span>
      <i></i>
      <span>ONE DECISION. ONE DAY. ONE TIDE.</span>
    </section>

    <section class="section features" id="gameplay">
      <div class="section-heading centered">
        <p class="kicker">THE DAILY RITUAL</p>
        <h2>A few meaningful choices.<br />A full day of consequences.</h2>
        <p>Tidekeepers turns public market movement into a shared fantasy battlefield. Your build—not a lucky prediction—decides how well you endure it.</p>
      </div>

      <div class="feature-grid">
        {#each features as feature}
          <article class="feature-card">
            <div class="feature-icon">{feature.icon}</div>
            <p class="mini-label">{feature.eyebrow}</p>
            <h3>{feature.title}</h3>
            <p>{feature.text}</p>
          </article>
        {/each}
      </div>
    </section>

    <section class="section split-section" id="voyage">
      <div class="split-copy">
        <p class="kicker">YOUR SEVEN-DAY VOYAGE</p>
        <h2>Every Tide asks a different question.</h2>
        <p>
          Broad Growth. Red Tide. Recovery Wave. Rotation Current. Each day reshapes the battlefield with a modifier, objective, and set of signals.
        </p>
        <p>
          Survive long enough to reach the Final Tide. Lose your Hull, and the Voyage ends.
        </p>
        <a class="text-link" href="#join">Answer the first Tide <span>→</span></a>
      </div>

      <div class="voyage-map" aria-label="Seven-day Voyage progression">
        <div class="map-line"></div>
        {#each Array(7) as _, index}
          <div class:active={index === 2} class:boss={index === 6} class="map-node">
            <span>{index === 6 ? '✺' : index + 1}</span>
            <small>{index === 6 ? 'Final Tide' : `Day ${index + 1}`}</small>
          </div>
        {/each}
      </div>
    </section>

    <section class="section keepers-section" id="keepers">
      <div class="section-heading">
        <p class="kicker">MEET THE KEEPERS</p>
        <h2>Your fleet is your answer to the Tide.</h2>
        <p>Each Keeper represents a different approach to growth, defense, information, and recovery.</p>
      </div>

      <div class="keeper-grid">
        {#each keepers as keeper}
          <article class="keeper-card {keeper.tone}">
            <div class="keeper-art">
              <img class="keeper-character" src={keeper.image} alt={`${keeper.name} character art`} />
              <span class="keeper-rune">✦</span>
            </div>
            <div class="keeper-copy">
              <p class="mini-label">{keeper.role}</p>
              <h3>{keeper.name}</h3>
              <p>{keeper.text}</p>
            </div>
          </article>
        {/each}
      </div>
    </section>

    <section class="section how-section">
      <div class="section-heading centered narrow">
        <p class="kicker">HOW A DAY UNFOLDS</p>
        <h2>Prepare now. Learn tomorrow.</h2>
      </div>
      <div class="steps">
        {#each steps as step}
          <article class="step">
            <span class="step-number">{step[0]}</span>
            <div>
              <h3>{step[1]}</h3>
              <p>{step[2]}</p>
            </div>
          </article>
        {/each}
      </div>
    </section>

    <section class="section quote-section">
      <div class="quote-mark">“</div>
      <blockquote>
        You do not need to predict tomorrow.<br />You need a fleet that can survive it.
      </blockquote>
      <p>— The Tidekeeper’s Creed</p>
    </section>

    <section class="cta-section" id="join">
      <div class="cta-glow"></div>
      <div class="cta-content">
        <p class="kicker">THE LOCK CLOSES AT DAWN</p>
        <h2>Will your line hold?</h2>
        <p>Begin a seven-day Voyage. Read the signals, shape your fleet, and face what tomorrow brings.</p>
        <a class="button button-primary button-large" href="/play/prepare">Begin Your Voyage</a>
        <small>Free to play · No wallet · No real-money trading</small>
      </div>
    </section>
  </main>

  <footer>
    <a class="brand footer-brand" href="#top"><span class="brand-mark">✦</span><span class="brand-word">TIDEKEEPERS</span></a>
    <p>Build today. Face tomorrow.</p>
    <div>
      <a href="/help">How to Play</a>
      <a href="/legal/privacy">Privacy</a>
      <a href="/legal/terms">Terms</a>
    </div>
  </footer>
</div>

<style>
  :global(*) { box-sizing: border-box; }
  :global(html) { scroll-behavior: smooth; }
  :global(body) {
    margin: 0;
    color: #eee1c6;
    background:
      radial-gradient(circle at 50% -10%, rgba(14, 108, 117, .2), transparent 32rem),
      linear-gradient(180deg, #020d13 0%, #03141a 38%, #020b10 100%);
    font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  }
  :global(a) { color: inherit; text-decoration: none; }
  :global(button), :global(a) { -webkit-tap-highlight-color: transparent; }

  .page-shell { overflow: hidden; min-height: 100vh; }
  .page-shell::before {
    content: '';
    position: fixed;
    inset: 0;
    pointer-events: none;
    opacity: .18;
    background-image:
      linear-gradient(rgba(75, 171, 175, .14) 1px, transparent 1px),
      linear-gradient(90deg, rgba(75, 171, 175, .12) 1px, transparent 1px);
    background-size: 54px 54px;
    mask-image: linear-gradient(to bottom, transparent, #000 28%, transparent 92%);
  }

  .site-header {
    width: min(1440px, calc(100% - 40px));
    height: 82px;
    margin: 0 auto;
    display: grid;
    grid-template-columns: 1fr auto 1fr;
    align-items: center;
    border-bottom: 1px solid rgba(176, 138, 70, .28);
    position: relative;
    z-index: 10;
  }
  .brand { display: inline-flex; align-items: center; gap: 14px; width: fit-content; }
  .brand-mark {
    width: 42px; height: 42px; border-radius: 50%; display: grid; place-items: center;
    border: 1px solid rgba(36, 205, 205, .48);
    background: radial-gradient(circle, rgba(45, 198, 190, .15), rgba(3, 17, 22, .7));
    color: #efe2c6; text-shadow: 0 0 18px rgba(91, 241, 234, .7);
  }
  .brand-word { font-family: Georgia, 'Times New Roman', serif; font-size: 25px; letter-spacing: .08em; }
  nav { display: flex; gap: 38px; font-family: Georgia, 'Times New Roman', serif; font-size: 16px; color: #cbbd9f; }
  nav a:hover, footer a:hover { color: #6ee1dc; }
  .site-header > .button { justify-self: end; }

  .button {
    display: inline-flex; align-items: center; justify-content: center;
    min-height: 52px; padding: 0 28px; border: 1px solid rgba(64, 212, 208, .55);
    font-size: 13px; font-weight: 700; letter-spacing: .13em; text-transform: uppercase;
    clip-path: polygon(10px 0, calc(100% - 10px) 0, 100% 10px, 100% calc(100% - 10px), calc(100% - 10px) 100%, 10px 100%, 0 calc(100% - 10px), 0 10px);
    transition: .25s ease;
  }
  .button:hover { transform: translateY(-2px); box-shadow: 0 14px 45px rgba(31, 197, 190, .18); }
  .button-primary { color: #041013; background: linear-gradient(180deg, #61ded8, #2ba9a6); border-color: #78f0e8; }
  .button-outline { color: #72d8d3; background: rgba(5, 30, 35, .55); }
  .button-small { min-height: 42px; padding-inline: 20px; }
  .button-large { min-width: 250px; min-height: 58px; }
  .text-link { display: inline-flex; gap: 12px; align-items: center; color: #75ded8; font-weight: 700; letter-spacing: .04em; }
  .text-link span { transition: transform .2s ease; }
  .text-link:hover span { transform: translateX(5px); }

  .section-frame, .section { width: min(1320px, calc(100% - 48px)); margin-inline: auto; }
  .hero { min-height: 760px; display: grid; grid-template-columns: .84fr 1.25fr; align-items: center; gap: 64px; padding: 82px 0 74px; position: relative; }
  .hero::after {
    content: ''; position: absolute; left: 45%; top: 18%; width: 52vw; height: 52vw; border-radius: 50%;
    border: 1px solid rgba(89, 205, 201, .11); box-shadow: 0 0 0 70px rgba(89, 205, 201, .025), 0 0 0 140px rgba(89, 205, 201, .02);
    pointer-events: none;
  }
  .hero-copy { position: relative; z-index: 2; }
  .kicker { margin: 0 0 16px; color: #57cfca; font-size: 12px; font-weight: 800; letter-spacing: .2em; }
  h1, h2, h3 { font-family: Georgia, 'Times New Roman', serif; font-weight: 500; }
  h1 { margin: 0; font-size: clamp(64px, 7vw, 112px); line-height: .88; letter-spacing: -.045em; }
  h1 span { color: #54d8d2; text-shadow: 0 0 34px rgba(70, 218, 211, .16); }
  .hero-lead { max-width: 620px; margin: 30px 0; color: #c4bda9; font-size: 20px; line-height: 1.65; }
  .hero-actions { display: flex; align-items: center; gap: 28px; flex-wrap: wrap; }
  .hero-stats { margin-top: 52px; display: flex; gap: 42px; }
  .hero-stats div { display: grid; gap: 4px; padding-left: 18px; border-left: 1px solid rgba(179, 139, 68, .42); }
  .hero-stats strong { font-family: Georgia, 'Times New Roman', serif; font-size: 20px; font-weight: 500; }
  .hero-stats span { color: #817e73; font-size: 12px; text-transform: uppercase; letter-spacing: .1em; }

  .hero-visual { position: relative; z-index: 2; min-width: 0; }
  .visual-glow { position: absolute; inset: 10% 8%; background: #21b9b2; filter: blur(110px); opacity: .14; }
  .dashboard-frame {
    position: relative; border: 1px solid rgba(159, 124, 58, .5); background: #03141b; padding: 8px;
    box-shadow: 0 45px 100px rgba(0, 0, 0, .48), 0 0 0 1px rgba(72, 218, 212, .08) inset;
    transform: perspective(1300px) rotateY(-7deg) rotateX(1deg);
  }
  .dashboard-frame::before, .dashboard-frame::after { content: ''; position: absolute; width: 54px; height: 54px; border-color: #4acac5; opacity: .75; }
  .dashboard-frame::before { left: -7px; top: -7px; border-left: 1px solid; border-top: 1px solid; }
  .dashboard-frame::after { right: -7px; bottom: -7px; border-right: 1px solid; border-bottom: 1px solid; }
  .dashboard-frame img { width: 100%; display: block; filter: saturate(.84) contrast(1.06); }
  .floating-card {
    position: absolute; z-index: 3; display: grid; gap: 3px; min-width: 190px; padding: 17px 20px;
    background: linear-gradient(135deg, rgba(4, 33, 40, .96), rgba(2, 15, 20, .96));
    border: 1px solid rgba(74, 211, 204, .35); box-shadow: 0 18px 50px rgba(0,0,0,.4);
  }
  .floating-card strong { font-family: Georgia, serif; font-size: 18px; font-weight: 500; }
  .floating-card small { color: #8e8b7e; }
  .floating-card-left { left: -28px; bottom: -34px; }
  .floating-card-right { right: -20px; top: -28px; }
  .mini-label { margin: 0; color: #4fcac5; font-size: 10px; font-weight: 800; letter-spacing: .16em; }

  .trust-strip {
    min-height: 72px; display: flex; align-items: center; justify-content: center; gap: 28px;
    border-block: 1px solid rgba(178, 135, 61, .24); color: #8f8879; font-size: 11px; letter-spacing: .14em; text-align: center;
    background: rgba(3, 17, 22, .78);
  }
  .trust-strip i { width: 5px; height: 5px; transform: rotate(45deg); border: 1px solid #2bbcb7; }

  .section { padding-block: 122px; }
  .section-heading { max-width: 740px; margin-bottom: 48px; }
  .section-heading.centered { text-align: center; margin-inline: auto; margin-bottom: 64px; }
  .section-heading.narrow { max-width: 620px; }
  h2 { margin: 0 0 22px; font-size: clamp(42px, 5vw, 70px); line-height: 1.05; letter-spacing: -.03em; }
  .section-heading > p:last-child, .split-copy > p, .feature-card > p:last-child, .keeper-copy p, .step p, .cta-content > p {
    color: #a8a293; line-height: 1.7;
  }
  .feature-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 18px; }
  .feature-card {
    min-height: 320px; padding: 36px; position: relative;
    border: 1px solid rgba(176, 137, 69, .28); background: linear-gradient(160deg, rgba(4, 29, 35, .92), rgba(2, 14, 19, .85));
  }
  .feature-card::after { content: ''; position: absolute; left: 24px; right: 24px; top: 0; height: 1px; background: linear-gradient(90deg, transparent, #41c9c3, transparent); opacity: .7; }
  .feature-icon { width: 62px; height: 62px; display: grid; place-items: center; margin-bottom: 48px; border: 1px solid rgba(58, 202, 195, .48); border-radius: 50%; color: #64ded8; font-size: 25px; }
  .feature-card h3 { margin: 10px 0 16px; font-size: 29px; }

  .split-section { display: grid; grid-template-columns: .82fr 1.18fr; gap: 86px; align-items: center; }
  .split-copy { max-width: 560px; }
  .split-copy > p { font-size: 17px; }
  .voyage-map { min-height: 420px; display: grid; grid-template-columns: repeat(7, 1fr); align-items: center; position: relative; }
  .map-line { position: absolute; left: 7%; right: 7%; top: 48%; border-top: 1px dashed rgba(78, 200, 194, .46); }
  .map-node { position: relative; display: grid; place-items: center; gap: 14px; color: #938c7a; }
  .map-node span { width: 46px; height: 46px; border-radius: 50%; display: grid; place-items: center; border: 1px solid rgba(173, 137, 70, .55); background: #04151b; z-index: 1; }
  .map-node small { font-family: Georgia, serif; white-space: nowrap; }
  .map-node.active span { width: 76px; height: 76px; color: #cffffb; border-color: #5ee5dd; box-shadow: 0 0 35px rgba(64, 221, 213, .22); }
  .map-node.active small { color: #5ee5dd; font-size: 15px; }
  .map-node.boss span { border-color: #c89745; color: #e2ad56; }

  .keepers-section { border-top: 1px solid rgba(176, 137, 69, .2); }
  .keeper-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; }
  .keeper-card { overflow: hidden; border: 1px solid rgba(171, 134, 70, .28); background: #03151b; }
  .keeper-art { height: 315px; position: relative; display: grid; place-items: end center; overflow: hidden; background: radial-gradient(circle at 50% 30%, rgba(38, 159, 157, .28), transparent 48%), linear-gradient(180deg, #06232a, #020d12); }
  .keeper-art::before { content: ''; position: absolute; inset: 18px; border: 1px solid rgba(108, 210, 203, .15); pointer-events: none; }
  .keeper-character { position: absolute; inset: 10px; width: calc(100% - 20px); height: calc(100% - 10px); object-fit: contain; object-position: center bottom; filter: saturate(.9) drop-shadow(0 0 28px rgba(67, 213, 206, .22)); }
  .keeper-rune { position: absolute; top: 24px; right: 24px; z-index: 1; color: #71dfd9; }
  .keeper-copy { min-height: 188px; padding: 28px; }
  .keeper-copy h3 { margin: 9px 0 12px; font-size: 27px; }
  .keeper-card.harbor .keeper-art { background: radial-gradient(circle at 50% 30%, rgba(37, 147, 125, .29), transparent 48%), linear-gradient(180deg, #07241f, #020d12); }
  .keeper-card.echo .keeper-art { background: radial-gradient(circle at 50% 30%, rgba(73, 90, 192, .3), transparent 48%), linear-gradient(180deg, #0a1731, #020d12); }
  .keeper-card.current .keeper-art { background: radial-gradient(circle at 50% 30%, rgba(35, 168, 184, .33), transparent 48%), linear-gradient(180deg, #05242b, #020d12); }

  .how-section { display: grid; grid-template-columns: .72fr 1.28fr; gap: 90px; align-items: start; }
  .how-section .section-heading { position: sticky; top: 100px; text-align: left; margin: 0; }
  .steps { border-top: 1px solid rgba(172, 135, 68, .3); }
  .step { display: grid; grid-template-columns: 78px 1fr; gap: 24px; padding: 34px 0; border-bottom: 1px solid rgba(172, 135, 68, .24); }
  .step-number { color: #50d2cc; font-size: 13px; font-weight: 800; letter-spacing: .12em; }
  .step h3 { margin: 0 0 10px; font-size: 29px; }
  .step p { margin: 0; }

  .quote-section { text-align: center; padding-block: 152px; border-block: 1px solid rgba(172, 135, 68, .2); }
  .quote-mark { height: 52px; color: #47c9c3; font-family: Georgia, serif; font-size: 80px; line-height: 1; }
  blockquote { margin: 24px auto; max-width: 980px; font-family: Georgia, serif; font-size: clamp(38px, 5vw, 68px); line-height: 1.16; }
  .quote-section > p { color: #7e7a70; letter-spacing: .12em; text-transform: uppercase; font-size: 11px; }

  .cta-section { min-height: 610px; display: grid; place-items: center; position: relative; overflow: hidden; border-bottom: 1px solid rgba(174, 134, 66, .24); background: radial-gradient(circle at 50% 60%, rgba(35, 186, 179, .16), transparent 34%), linear-gradient(180deg, rgba(2, 12, 17, .7), #021016); }
  .cta-section::before { content: ''; position: absolute; width: 620px; height: 620px; border: 1px solid rgba(71, 201, 195, .17); border-radius: 50%; box-shadow: 0 0 0 80px rgba(71, 201, 195, .025), 0 0 0 160px rgba(71, 201, 195, .018); }
  .cta-content { position: relative; z-index: 1; width: min(680px, calc(100% - 40px)); text-align: center; }
  .cta-content h2 { font-size: clamp(54px, 7vw, 92px); }
  .cta-content > p { font-size: 18px; }
  .cta-content .button { margin: 20px auto 16px; }
  .cta-content small { display: block; color: #77756c; }

  footer { width: min(1320px, calc(100% - 48px)); min-height: 150px; margin: 0 auto; display: grid; grid-template-columns: 1fr auto 1fr; align-items: center; color: #77756c; font-size: 13px; }
  .footer-brand { opacity: .86; }
  footer > p { text-align: center; font-family: Georgia, serif; }
  footer > div { justify-self: end; display: flex; gap: 24px; }

  @media (max-width: 1100px) {
    .site-header { grid-template-columns: 1fr auto; }
    nav { display: none; }
    .hero { grid-template-columns: 1fr; padding-top: 74px; }
    .hero-copy { max-width: 760px; }
    .hero-visual { margin-top: 20px; }
    .feature-grid { grid-template-columns: 1fr; }
    .feature-card { min-height: auto; }
    .split-section, .how-section { grid-template-columns: 1fr; }
    .how-section .section-heading { position: static; }
    .keeper-grid { grid-template-columns: repeat(2, 1fr); }
  }

  @media (max-width: 720px) {
    .site-header { width: calc(100% - 28px); height: 70px; }
    .brand-word { font-size: 18px; }
    .brand-mark { width: 36px; height: 36px; }
    .site-header > .button { display: none; }
    .section-frame, .section { width: calc(100% - 32px); }
    .hero { min-height: auto; padding: 64px 0 70px; gap: 40px; }
    h1 { font-size: clamp(56px, 18vw, 78px); }
    .hero-lead { font-size: 17px; }
    .hero-stats { gap: 16px; justify-content: space-between; }
    .hero-stats div { padding-left: 10px; }
    .hero-stats strong { font-size: 15px; }
    .hero-stats span { font-size: 9px; }
    .dashboard-frame { transform: none; padding: 4px; }
    .floating-card { display: none; }
    .trust-strip { padding: 22px 18px; flex-wrap: wrap; gap: 12px; }
    .trust-strip i { display: none; }
    .section { padding-block: 88px; }
    h2 { font-size: 44px; }
    .feature-card { padding: 28px; }
    .voyage-map { min-height: 280px; overflow-x: auto; grid-template-columns: repeat(7, 90px); justify-content: start; }
    .map-line { left: 45px; right: auto; width: 540px; }
    .keeper-grid { grid-template-columns: 1fr; }
    .keeper-art { height: 270px; }
    .step { grid-template-columns: 48px 1fr; }
    blockquote { font-size: 38px; }
    footer { grid-template-columns: 1fr; gap: 20px; padding-block: 42px; text-align: center; }
    footer .brand, footer > div { justify-self: center; }
  }
</style>
