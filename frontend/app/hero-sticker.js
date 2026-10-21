import styles from "./hero-sticker.module.css";

// Все анимации — CSS-циклы: кадр 0% равен кадру 100%, периоды делят 12 с.
const INK = "#1b1b1f";
const YELLOW = "#ffc61a";
const RED = "#e8452c";
const TEAL = "#2ba6b5";
const CREAM = "#f4ecd2";

const c = (...names) => names.map((name) => styles[name]).join(" ");

function Defs() {
  return (
    <defs>
      <pattern id="ht" width="5" height="5" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
        <circle cx="2.5" cy="2.5" r="1.35" fill={INK} opacity="0.6" />
      </pattern>
      <pattern id="htw" width="6" height="6" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
        <circle cx="3" cy="3" r="1.5" fill="#fff" opacity="0.28" />
      </pattern>
      <pattern id="htr" width="7" height="7" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
        <circle cx="3.5" cy="3.5" r="1.9" fill={RED} opacity="0.45" />
      </pattern>
      <filter id="sticker" x="-15%" y="-15%" width="130%" height="130%">
        <feMorphology in="SourceAlpha" operator="dilate" radius="4" result="grow" />
        <feFlood floodColor="#fff" />
        <feComposite in2="grow" operator="in" result="edge" />
        <feMerge>
          <feMergeNode in="edge" />
          <feMergeNode in="SourceGraphic" />
        </feMerge>
      </filter>
    </defs>
  );
}

const CRATERS = [
  [-0.35, -0.45, 0.16, 0.22],
  [0.3, -0.5, 0.12, 0.1],
  [-0.62, 0.1, 0.1, 0.16],
  [0.08, 0.02, 0.2, 0.16],
  [0.52, 0.18, 0.12, 0.2],
  [-0.2, 0.55, 0.17, 0.12],
];

function Moon({ cx, cy, r, id }) {
  return (
    <g>
      <clipPath id={id}><circle cx={cx} cy={cy} r={r} /></clipPath>
      <circle cx={cx} cy={cy} r={r} fill={CREAM} stroke={INK} strokeWidth="5" />
      <g clipPath={`url(#${id})`}>
        <circle cx={cx + r * 0.55} cy={cy + r * 0.5} r={r} fill="url(#ht)" />
      </g>
      {CRATERS.map(([x, y, rx, ry], i) => (
        <g key={i}>
          <ellipse cx={cx + x * r} cy={cy + y * r} rx={rx * r} ry={ry * r} fill={TEAL} stroke={INK} strokeWidth="4" />
          <ellipse cx={cx + x * r} cy={cy + y * r + ry * r * 0.35} rx={rx * r * 0.8} ry={ry * r * 0.45} fill="url(#ht)" />
        </g>
      ))}
    </g>
  );
}

function Star({ x, y, s = 1, fill = YELLOW, delay = 0, period = "1.5s" }) {
  return (
    <g transform={`translate(${x} ${y}) scale(${s})`}>
      <g className={c("twinkle")} style={{ animationDelay: `${-delay}s`, animationDuration: period }}>
        <path d="M0,-12 L3.5,-3.5 L12,0 L3.5,3.5 L0,12 L-3.5,3.5 L-12,0 L-3.5,-3.5 Z" fill={fill} stroke={INK} strokeWidth="3" strokeLinejoin="round" />
      </g>
    </g>
  );
}

function Note({ x, y, delay = 0, fill = YELLOW }) {
  return (
    <g transform={`translate(${x} ${y})`}>
      <g className={c("noteUp")} style={{ animationDelay: `${-delay}s` }}>
        <path d="M0,0 V-26 Q12,-22 12,-10" fill="none" stroke={INK} strokeWidth="9" strokeLinecap="round" />
        <ellipse cx="-4" cy="2" rx="8" ry="6" transform="rotate(-20 -4 2)" fill={INK} />
        <path d="M0,0 V-26 Q12,-22 12,-10" fill="none" stroke={fill} strokeWidth="4" strokeLinecap="round" />
        <ellipse cx="-4" cy="2" rx="5" ry="3.5" transform="rotate(-20 -4 2)" fill={fill} />
      </g>
    </g>
  );
}

// Контур рисуется толстой обводкой под заливкой: так фигура из нескольких кругов получает один общий край.
function Figure({ fill, shape, fluffy = false, children }) {
  return (
    <g>
      <g fill={fill} stroke={INK} strokeWidth="8" strokeLinejoin="round">{shape}</g>
      <g fill={fill}>{shape}</g>
      {fluffy ? <g fill="url(#ht)" opacity="0.22">{shape}</g> : null}
      {children}
    </g>
  );
}

function Pill({ x, y, w, h, r = 0, fill = "#fff" }) {
  return <rect x={x - w / 2} y={y - h / 2} width={w} height={h} rx={w / 2} transform={`rotate(${r} ${x} ${y})`} fill={fill} />;
}

const circles = (list) => list.map(([x, y, r], i) => <circle key={i} cx={x} cy={y} r={r} />);

// Grok bots: плоские фигуры с белыми глазами-пилюлями, центр в (0,0), радиус около 26.
const GROK = [
  {
    fill: "#7a7a7a",
    shape: <path d="M8,-26 Q13,-26 18,-18 L24,-8 Q30,2 26,14 Q20,28 2,28 Q-22,28 -24,6 Q-24,-6 -14,-12 Z" />,
    eyes: [[-7, 4, 7, 15, 8], [7, 2, 7, 15, 8]],
  },
  {
    fill: "#1287ff",
    shape: <path d="M0,-26 Q7,-26 13,-15 L27,10 Q33,24 15,24 L-15,24 Q-33,24 -27,10 L-13,-15 Q-7,-26 0,-26 Z" />,
    eyes: [[-6, 11, 5, 11, 0], [6, 11, 5, 11, 0]],
  },
  {
    fill: "#ff6a00",
    shape: <>{circles([[-11, -8, 14], [6, -11, 14], [15, 5, 13], [3, 12, 12], [-14, 8, 12]])}</>,
    eyes: [[-5, -10, 5, 14, -70], [9, -5, 5, 14, -65]],
  },
  {
    fill: "#000",
    shape: <circle r="25" />,
    eyes: [[-10, -10, 4.5, 12, 25], [2, -8, 5, 12, 25]],
  },
  {
    fill: "#00c96f",
    shape: <rect x="-30" y="-19" width="60" height="38" rx="19" />,
    eyes: [[-8, -5, 8, 15, 0], [8, -5, 8, 15, 0]],
  },
  {
    fill: "#956439",
    shape: <g transform="rotate(25)">{circles([[-10, -10, 13], [10, -10, 13], [10, 10, 13], [-10, 10, 13], [0, 0, 12]])}</g>,
    eyes: [[-6, 2, 5, 12, -25], [6, -4, 5, 12, -25]],
  },
];

function Grok({ i }) {
  const bot = GROK[i % GROK.length];
  return (
    <Figure fill={bot.fill} shape={bot.shape}>
      {bot.eyes.map(([x, y, w, h, r], k) => <Pill key={k} x={x} y={y} w={w} h={h} r={r} />)}
    </Figure>
  );
}

// OpenAI dots: пушистые фигуры с аксессуарами, центр в (0,0).
const DOTS = [
  {
    fill: "#1e7bff",
    shape: <>{circles([[-12, 4, 17], [12, 4, 17], [0, -6, 18], [-20, 14, 11], [20, 14, 11]])}</>,
    face: (
      <>
        <ellipse cx="-7" cy="5" rx="3" ry="4.5" fill={INK} />
        <ellipse cx="7" cy="5" rx="3" ry="4.5" fill={INK} />
        <g transform="rotate(-10)">
          <ellipse cx="0" cy="-21" rx="24" ry="9" fill={INK} stroke="#fff" strokeWidth="1.5" />
          <circle cx="0" cy="-31" r="4.5" fill={INK} />
        </g>
      </>
    ),
  },
  {
    fill: "#8bd820",
    shape: <>{circles([[-12, -10, 11], [12, -10, 11]])}<ellipse cx="0" cy="8" rx="25" ry="18" /></>,
    face: (
      <>
        <circle cx="-12" cy="-11" r="7" fill="#fff" stroke={INK} strokeWidth="2" />
        <circle cx="12" cy="-11" r="7" fill="#fff" stroke={INK} strokeWidth="2" />
        <circle cx="-11" cy="-10" r="3.5" fill={INK} />
        <circle cx="13" cy="-10" r="3.5" fill={INK} />
      </>
    ),
  },
  {
    fill: "#ffd21f",
    shape: <path d="M0,-26 Q8,-26 14,-12 L26,16 Q30,26 18,26 L-18,26 Q-30,26 -26,16 L-14,-12 Q-8,-26 0,-26 Z" />,
    face: (
      <g fill="none" stroke={INK} strokeLinecap="round">
        <circle cx="-9" cy="6" r="8" strokeWidth="2.5" />
        <circle cx="9" cy="6" r="8" strokeWidth="2.5" />
        <path d="M-1,6 H1 M-17,5 L-24,3 M17,5 L24,3" strokeWidth="2.5" />
        <path d="M-13,6 Q-9,10 -5,6 M5,6 Q9,10 13,6" strokeWidth="2" />
      </g>
    ),
  },
  {
    fill: "#ff3db4",
    shape: <path d="M0,26 L-24,2 Q-34,-12 -24,-22 Q-12,-30 0,-16 Q12,-30 24,-22 Q34,-12 24,2 Z" />,
    face: (
      <>
        <path d="M-26,-4 H26" stroke={INK} strokeWidth="2.5" />
        <circle cx="-10" cy="-4" r="7" fill={INK} />
        <circle cx="10" cy="-4" r="7" fill={INK} />
      </>
    ),
  },
];

function Dot({ i }) {
  const dot = DOTS[i % DOTS.length];
  return <Figure fill={dot.fill} shape={dot.shape} fluffy>{dot.face}</Figure>;
}

const MUSE_BODY = "M-32,40 Q-37,4 -32,-22 Q-25,-48 0,-48 Q25,-48 32,-22 Q37,4 32,40 Q0,47 -32,40 Z";

// Meta Muse: кремовый пушистый капюшон с лицом, центр в (0,0); шлем — по умолчанию.
function Muse({ helmet = true }) {
  return (
    <g>
      <Figure
        fill="#f1e6cf"
        fluffy
        shape={
          <>
            <ellipse cx="-33" cy="12" rx="9" ry="15" transform="rotate(18 -33 12)" />
            <ellipse cx="33" cy="12" rx="9" ry="15" transform="rotate(-18 33 12)" />
            <path d={MUSE_BODY} />
          </>
        }
      />
      <ellipse cx="0" cy="-16" rx="22" ry="17" fill="#f9dcc4" stroke={INK} strokeWidth="3" />
      <circle cx="-13" cy="-10" r="4.5" fill="#f2a0a0" />
      <circle cx="13" cy="-10" r="4.5" fill="#f2a0a0" />
      <circle cx="-9" cy="-18" r="3.6" fill={INK} />
      <circle cx="9" cy="-18" r="3.6" fill={INK} />
      <circle cx="-8" cy="-19.2" r="1.1" fill="#fff" />
      <circle cx="10" cy="-19.2" r="1.1" fill="#fff" />
      <path d="M-5,-10 Q0,-6 5,-10" fill="none" stroke={INK} strokeWidth="2.2" strokeLinecap="round" />
      {helmet ? (
        <>
          <circle cx="0" cy="-14" r="38" fill={TEAL} fillOpacity="0.16" stroke={INK} strokeWidth="4" />
          <path d="M-24,-38 Q-12,-50 6,-50" fill="none" stroke="#fff" strokeWidth="5" strokeLinecap="round" />
          <rect x="-28" y="18" width="56" height="9" rx="4" fill={YELLOW} stroke={INK} strokeWidth="3.5" />
        </>
      ) : null}
    </g>
  );
}

function Flame({ x }) {
  return (
    <g transform={`translate(${x} 0)`}>
      <g className={c("flame")}>
        <path d="M0,0 L-52,-20 L-34,-7 L-66,0 L-34,7 L-52,20 Z" fill={RED} stroke={INK} strokeWidth="4" strokeLinejoin="round" />
        <path d="M-4,0 L-30,-9 L-20,0 L-30,9 Z" fill={YELLOW} />
      </g>
    </g>
  );
}

// Ракета с Muse в кабине, нос вправо, центр в (0,0).
function Rocket() {
  return (
    <g>
      <Flame x={-86} />
      <path d="M-62,-18 L-96,-52 L-30,-26 Z" fill={RED} stroke={INK} strokeWidth="5" strokeLinejoin="round" />
      <path d="M-62,18 L-96,52 L-30,26 Z" fill={RED} stroke={INK} strokeWidth="5" strokeLinejoin="round" />
      <g transform="translate(-8 -50) scale(0.78)">
        <g className={c("astro")}><Muse /></g>
      </g>
      <path d="M-84,-22 L20,-28 Q75,-20 102,0 Q75,20 20,28 L-84,22 Z" fill={YELLOW} stroke={INK} strokeWidth="5" strokeLinejoin="round" />
      <path d="M-84,10 L20,16 Q60,12 90,6 Q75,20 20,28 L-84,22 Z" fill="url(#ht)" />
      <path d="M58,-22 Q88,-14 102,0 Q88,14 58,22 Q66,0 58,-22 Z" fill={RED} stroke={INK} strokeWidth="5" strokeLinejoin="round" />
      <rect x="-58" y="-26" width="14" height="52" fill={RED} stroke={INK} strokeWidth="4" />
      <circle cx="18" cy="4" r="11" fill={TEAL} stroke={INK} strokeWidth="4" />
      <circle cx="18" cy="8" r="6" fill="url(#ht)" />
    </g>
  );
}

const PORTS = [-92, -40, 12, 64];
const DECK = [-112, -76, -40, -4, 32, 68];

// Длинный крейсер: Dots в иллюминаторах, Grok bots на крыше. Нос вправо, центр в (0,0).
function Cruiser({ id }) {
  return (
    <g>
      <Flame x={-140} />
      <path d="M-110,-24 L-146,-62 L-70,-30 Z" fill={RED} stroke={INK} strokeWidth="5" strokeLinejoin="round" />
      <path d="M-110,24 L-146,62 L-70,30 Z" fill={RED} stroke={INK} strokeWidth="5" strokeLinejoin="round" />
      {DECK.map((x, i) => (
        <g key={i} transform={`translate(${x} -46) scale(0.58)`}>
          <g className={c("jitter")} style={{ animationDelay: `${-i * 0.25}s` }}><Grok i={i} /></g>
        </g>
      ))}
      <path d="M-138,-30 L80,-34 Q140,-26 172,0 Q140,26 80,34 L-138,30 Z" fill={YELLOW} stroke={INK} strokeWidth="5" strokeLinejoin="round" />
      <path d="M-138,14 L80,18 Q130,14 160,8 Q140,26 80,34 L-138,30 Z" fill="url(#ht)" />
      <path d="M120,-29 Q150,-18 172,0 Q150,18 120,29 Q130,0 120,-29 Z" fill={RED} stroke={INK} strokeWidth="5" strokeLinejoin="round" />
      <rect x="-128" y="-30" width="14" height="60" fill={RED} stroke={INK} strokeWidth="4" />
      {PORTS.map((x, i) => (
        <g key={i} transform={`translate(${x} 0)`}>
          <clipPath id={`${id}${i}`}><circle r="20" /></clipPath>
          <circle r="20" fill={TEAL} />
          <circle r="20" fill="url(#htw)" />
          <g clipPath={`url(#${id}${i})`}>
            <g transform="translate(0 8) scale(0.6)">
              <g className={c("astro")} style={{ animationDelay: `${-i * 0.75}s` }}><Dot i={i} /></g>
            </g>
          </g>
          <circle r="20" fill="none" stroke={INK} strokeWidth="5" />
          <path d="M-12,-10 Q-6,-15 2,-15" fill="none" stroke="#fff" strokeWidth="3.5" strokeLinecap="round" />
        </g>
      ))}
    </g>
  );
}

function Panel({ w, h, children }) {
  return (
    <>
      <clipPath id="panelclip"><rect x="6" y="6" width={w - 12} height={h - 12} rx="26" /></clipPath>
      <rect x="6" y="6" width={w - 12} height={h - 12} rx="26" fill={RED} />
      <g clipPath="url(#panelclip)">
        <rect x="6" y="6" width={w - 12} height={h - 12} fill="url(#htw)" />
        {children}
      </g>
      <rect x="6" y="6" width={w - 12} height={h - 12} rx="26" fill="none" stroke={INK} strokeWidth="6" />
    </>
  );
}

function Scroll({ w, speed = "6s", children }) {
  return (
    <g className={c("scroll")} style={{ "--w": `${w}px`, animationDuration: speed }}>
      {children}
      <g transform={`translate(${w} 0)`}>{children}</g>
    </g>
  );
}

function Svg({ w, h, className, children }) {
  return (
    <svg className={className} viewBox={`0 0 ${w} ${h}`} aria-hidden="true" focusable="false">
      <Defs />
      <g filter="url(#sticker)">{children}</g>
    </svg>
  );
}

// Комикс: сдвиг анимации так, чтобы её кадр 0% пришёлся на секунду t цикла period.
const at = (t, period = 6) => ({ animationDelay: `${(t % period) - period}s`, animationDuration: `${period}s` });

const round = (n) => Math.round(n * 10) / 10;

function ComicPanel({ id, d, fill, dots, children }) {
  return (
    <g>
      <clipPath id={id}><path d={d} /></clipPath>
      <path d={d} fill={fill} />
      <g clipPath={`url(#${id})`}>
        <path d={d} fill={`url(#${dots})`} />
        <g className={c("drift")}>{children}</g>
      </g>
      <path d={d} fill="none" stroke={INK} strokeWidth="6" strokeLinejoin="round" />
    </g>
  );
}

function SpeedLines({ cx, cy, n = 32, r1 = 56, r2 = 260, color = INK }) {
  return (
    <g stroke={color} strokeLinecap="round" opacity="0.5">
      {Array.from({ length: n }, (_, i) => {
        const a = (i / n) * Math.PI * 2 + (i % 2) * 0.06;
        const r = r1 + (i % 3) * 14;
        return (
          <line
            key={i}
            x1={round(cx + Math.cos(a) * r)}
            y1={round(cy + Math.sin(a) * r)}
            x2={round(cx + Math.cos(a) * r2)}
            y2={round(cy + Math.sin(a) * r2)}
            strokeWidth={i % 3 ? 1.8 : 3.5}
          />
        );
      })}
    </g>
  );
}

const burst = (r, n = 13) =>
  Array.from({ length: n * 2 }, (_, i) => {
    const a = (i / (n * 2)) * Math.PI * 2;
    const k = i % 2 ? 0.64 : 1 - (i % 3) * 0.07;
    return `${round(Math.cos(a) * r * k)},${round(Math.sin(a) * r * k)}`;
  }).join(" ");

function Word({ size, fill = YELLOW, rot = -8, children }) {
  return (
    <text
      transform={`rotate(${rot})`}
      fontFamily="'Arial Black', Impact, sans-serif"
      fontStyle="italic"
      fontWeight="900"
      fontSize={size}
      textAnchor="middle"
      dominantBaseline="central"
      fill={fill}
      stroke={INK}
      strokeWidth={size / 6}
      strokeLinejoin="round"
      paintOrder="stroke"
    >
      {children}
    </text>
  );
}

// Звукоподражание во вспышке: появляется в секунду t и гаснет.
function Bang({ x, y, r, t, period = 6, fill = YELLOW, ink = RED, size, rot, text }) {
  return (
    <g transform={`translate(${x} ${y})`}>
      <g className={c("pop")} style={at(t - 0.1, period)}>
        <polygon points={burst(r)} fill={fill} stroke={INK} strokeWidth="4" strokeLinejoin="round" />
        <Word size={size} fill={ink} rot={rot}>{text}</Word>
      </g>
    </g>
  );
}

function Caption({ x, y, children }) {
  const w = children.length * 11 + 16;
  return (
    <g transform={`translate(${x} ${y}) rotate(-2)`}>
      <rect width={w} height="24" fill={CREAM} stroke={INK} strokeWidth="3.5" />
      <text x={w / 2} y="13" fontFamily="'Arial Black', Impact, sans-serif" fontSize="13" textAnchor="middle" dominantBaseline="central" fill={INK}>{children}</text>
    </g>
  );
}

function Bubble({ x, y, w, h, tx, ty, children }) {
  return (
    <g>
      <Figure
        fill="#fff"
        shape={
          <>
            <ellipse cx={x} cy={y} rx={w / 2} ry={h / 2} />
            <path d={`M${x - 8},${y + h / 4} L${tx},${ty} L${x + 10},${y + h / 4} Z`} />
          </>
        }
      />
      <text x={x} y={y} fontFamily="'Arial Black', Impact, sans-serif" fontStyle="italic" fontSize="17" textAnchor="middle" dominantBaseline="central" fill={INK}>{children}</text>
    </g>
  );
}

// Бумажный самолётик — переносимый конфиг. Нос вправо, центр в (0,0).
function Plane() {
  return (
    <g stroke={INK} strokeWidth="3.5" strokeLinejoin="round">
      <path d="M24,0 L-20,-14 L-8,0 Z" fill="#fff" />
      <path d="M24,0 L-20,12 L-8,0 Z" fill={CREAM} />
      <path d="M24,0 L-6,4 L-14,16 Z" fill="#d9d2bc" />
    </g>
  );
}

function Card({ label }) {
  return (
    <g>
      <path d="M-15,-19 H7 L15,-11 V19 H-15 Z" fill="#fff" stroke={INK} strokeWidth="3.5" strokeLinejoin="round" />
      <path d="M7,-19 V-11 H15" fill="none" stroke={INK} strokeWidth="3" strokeLinejoin="round" />
      <path d="M-9,-8 H3 M-9,-2 H9" stroke={INK} strokeWidth="2.5" strokeLinecap="round" />
      <text y="10" fontFamily="'Arial Black', Impact, sans-serif" fontSize="9" textAnchor="middle" dominantBaseline="central" fill={RED}>{label}</text>
    </g>
  );
}

function Key() {
  return (
    <g fill={YELLOW} stroke={INK} strokeWidth="4" strokeLinejoin="round">
      <path d="M-4,-5 H30 V3 H26 V11 H19 V3 H14 V8 H8 V3 H-4 Z" />
      <circle cx="-12" cy="-1" r="12" />
      <circle cx="-12" cy="-1" r="4" fill={INK} stroke="none" />
      <circle cx="-15" cy="-5" r="3" fill="#fff" stroke="none" />
    </g>
  );
}

// Ноутбук, центр основания в (0,0); children рисуются на экране с центром в (0,-40).
function Laptop({ children }) {
  return (
    <g>
      <rect x="-50" y="-74" width="100" height="68" rx="7" fill="#3b3b45" stroke={INK} strokeWidth="4" />
      <rect x="-41" y="-66" width="82" height="52" rx="3" fill={TEAL} />
      <rect x="-41" y="-66" width="82" height="52" rx="3" fill="url(#htw)" />
      <path d="M-30,-54 H6 M-30,-44 H18 M-22,-34 H10" stroke={INK} strokeWidth="3" strokeLinecap="round" opacity="0.5" />
      {children}
      <path d="M-62,-8 H62 L72,8 H-72 Z" fill="#c9ced6" stroke={INK} strokeWidth="4" strokeLinejoin="round" />
    </g>
  );
}

const RELAY_BOTS = [[0, 42, 214, 0.9], [1, 86, 216, 1.6]];
const BEAM_CAST = [<Grok key="g1" i={1} />, <Dot key="d3" i={3} />, <Grok key="g2" i={2} />, <Dot key="d0" i={0} />];

// 18 Grok bots через 20°: узор повторяется каждые 120°, поэтому поворот на 120° за цикл бесшовен.
const RING = Array.from({ length: 18 }, (_, i) => i);
const DOTS_ON_MOON = [[104, 145], [148, 134], [192, 134], [236, 145]];

const VARIANTS = {
  // А — «Лунный старт»: Muse летит на ракете; справа Dots прыгают на луне, над ней по орбите идут Grok bots.
  a: {
    title: "А · Лунный старт",
    left: (cls) => (
      <Svg w={200} h={200} className={cls}>
        <g className={c("float")} style={{ animationDuration: "6s" }}>
          <Moon cx={130} cy={90} r={62} id="mA" />
        </g>
        <Star x={26} y={30} delay={0} />
        <Star x={182} y={176} s={0.8} delay={0.6} fill={RED} />
        <Star x={30} y={170} s={0.6} delay={1.1} fill={TEAL} />
        <g transform="translate(98 128) rotate(-24) scale(0.62)">
          <g className={c("fly")}><Rocket /></g>
        </g>
      </Svg>
    ),
    right: (cls) => (
      <Svg w={340} h={260} className={cls}>
        <Star x={30} y={36} delay={0.2} />
        <Star x={310} y={30} s={0.8} delay={0.9} fill={TEAL} />
        <Star x={300} y={150} s={0.7} delay={0.5} fill={RED} />
        <g transform="translate(170 340)">
          <g className={c("orbit")}>
            <circle r="290" fill="none" />
            {RING.map((i) => (
              <g key={i} transform={`rotate(${i * 20}) translate(0 -260) rotate(${-i * 20}) scale(0.62)`}>
                <g className={c("counterOrbit")}>
                  <circle r="34" fill="none" />
                  <Grok i={i} />
                </g>
              </g>
            ))}
          </g>
        </g>
        <Moon cx={170} cy={340} r={185} id="mA2" />
        {DOTS_ON_MOON.map(([x, y], i) => (
          <g key={i} transform={`translate(${x} ${y}) scale(0.8)`}>
            <g className={c("hop")} style={{ animationDelay: `${-i * 0.375}s` }}><Dot i={i} /></g>
          </g>
        ))}
      </Svg>
    ),
  },
  // Б — «Крейсер»: Muse парит в шлеме; справа крейсер стоит в кадре, мимо плывёт параллакс.
  b: {
    title: "Б · Крейсер",
    left: (cls) => (
      <Svg w={200} h={200} className={cls}>
        <g transform="translate(100 104)">
          <g className={c("spin")} style={{ animationDuration: "12s" }}>
            <ellipse cx="0" cy="0" rx="92" ry="26" fill="none" stroke={INK} strokeWidth="14" transform="rotate(-22)" />
            <ellipse cx="0" cy="0" rx="92" ry="26" fill="none" stroke={YELLOW} strokeWidth="7" transform="rotate(-22)" />
          </g>
          <g className={c("float")}>
            <g className={c("sway")}><Muse /></g>
          </g>
          <g className={c("spin")} style={{ animationDuration: "6s" }}>
            <circle r="84" fill="none" />
            <g transform="translate(84 0)">
              <g className={c("counter")}>
                <circle r="13" fill={CREAM} stroke={INK} strokeWidth="4" />
                <circle cx="4" cy="4" r="7" fill="url(#ht)" />
              </g>
            </g>
          </g>
        </g>
        <Star x={26} y={28} delay={0.2} />
        <Star x={176} y={170} s={0.8} delay={0.9} fill={RED} />
      </Svg>
    ),
    right: (cls) => (
      <Svg w={340} h={260} className={cls}>
        <Panel w={340} h={260}>
          <Scroll w={340} speed="12s">
            {[[40, 46], [120, 220], [200, 40], [270, 200], [310, 90]].map(([x, y], i) => (
              <circle key={i} cx={x} cy={y} r="3.5" fill={CREAM} />
            ))}
          </Scroll>
          <Scroll w={340} speed="6s">
            <Moon cx={250} cy={236} r={56} id="mB" />
            <Star x={60} y={50} delay={0.4} />
            <Star x={150} y={228} s={0.8} delay={1} fill={TEAL} />
          </Scroll>
          <g className={c("trail")} style={{ animationDuration: "0.4s" }}>
            <path d="M-10,118 H60" stroke={CREAM} strokeWidth="6" strokeLinecap="round" strokeDasharray="26 40" />
            <path d="M-10,160 H50" stroke={CREAM} strokeWidth="6" strokeLinecap="round" strokeDasharray="26 40" />
          </g>
          <g transform="translate(182 140) rotate(-6) scale(0.8)">
            <g className={c("jitter")} style={{ animationDuration: "3s" }}><Cruiser id="pB" /></g>
          </g>
        </Panel>
      </Svg>
    ),
  },
  // В — «Шоу талантов»: Muse поёт на луне; справа вся труппа танцует в такт под надписью SYNC!
  c: {
    title: "В · Шоу талантов",
    left: (cls) => (
      <Svg w={200} h={200} className={cls}>
        <Moon cx={100} cy={236} r={84} id="mC" />
        <g transform="translate(100 108)">
          <g className={c("swing")}>
            <Muse />
            <path d="M26,8 L38,-14" stroke={INK} strokeWidth="9" strokeLinecap="round" />
            <path d="M26,8 L38,-14" stroke={RED} strokeWidth="4" strokeLinecap="round" />
            <circle cx="40" cy="-20" r="9" fill={TEAL} stroke={INK} strokeWidth="4" />
            <circle cx="42" cy="-18" r="6" fill="url(#ht)" />
          </g>
        </g>
        <Note x={40} y={96} delay={0} />
        <Note x={170} y={88} delay={1.5} fill={TEAL} />
        <Star x={176} y={130} delay={0.6} fill={RED} />
        <Star x={24} y={130} s={0.7} delay={1.2} />
      </Svg>
    ),
    right: (cls) => (
      <Svg w={340} h={260} className={cls}>
        <g transform="translate(170 150)">
          <g className={c("spin")} style={{ animationDuration: "12s" }}>
            {Array.from({ length: 12 }, (_, i) => (
              <path key={i} d="M0,0 L-20,-100 L20,-100 Z" transform={`rotate(${i * 30})`} fill={i % 2 ? YELLOW : TEAL} stroke={INK} strokeWidth="4" strokeLinejoin="round" />
            ))}
          </g>
          <circle r="104" fill="url(#ht)" opacity="0.35" />
        </g>
        <Moon cx={170} cy={430} r={210} id="mC2" />
        {GROK.map((_, i) => (
          <g key={i} transform={`translate(${60 + i * 44} 150) scale(0.6)`}>
            <g className={c("hop")} style={{ animationDelay: `${-i * 0.25}s` }}><Grok i={i} /></g>
          </g>
        ))}
        {DOTS.map((_, i) => (
          <g key={i} transform={`translate(${104 + i * 44} 198) scale(0.82)`}>
            <g className={c("beat")} style={{ animationDelay: `${-i * 0.375}s` }}><Dot i={i} /></g>
          </g>
        ))}
        <g transform="translate(170 40)">
          <rect x="-96" y="-30" width="192" height="58" rx="14" fill={INK} stroke="#fff" strokeWidth="3" />
          {[...("SYNC!")].map((ch, i) => (
            <text
              key={i}
              x={-72 + i * 36}
              y="16"
              className={c("letter")}
              style={{ animationDelay: `${-1.5 + i * 0.12}s` }}
              fontFamily="'Arial Black', Impact, sans-serif"
              fontStyle="italic"
              fontSize="50"
              textAnchor="middle"
              fill={YELLOW}
            >
              {ch}
            </text>
          ))}
        </g>
        <Star x={26} y={110} delay={0.2} />
        <Star x={316} y={110} s={0.9} delay={0.9} fill={RED} />
      </Svg>
    ),
  },
  // Г — «Эстафета»: Muse бросает самолётик-конфиг; справа он летит через три кадра: Grok bots пасуют, SYNC!, Dots ловят.
  d: {
    title: "Г · Комикс: эстафета",
    left: (cls) => (
      <Svg w={200} h={200} className={cls}>
        <ComicPanel id="dL" d="M10,14 L192,8 L186,190 L8,194 Z" fill={YELLOW} dots="htr">
          <SpeedLines cx={86} cy={124} />
          <Bang x={150} y={54} r={40} t={1.8} period={3} fill="#fff" size={19} text="ВЖУХ!" />
          <g transform="translate(86 132)">
            <g className={c("windup")}><Muse helmet={false} /></g>
          </g>
          <g transform="translate(124 140)">
            <g className={c("launch")}><Plane /></g>
          </g>
        </ComicPanel>
      </Svg>
    ),
    right: (cls) => (
      <Svg w={340} h={260} className={cls}>
        <ComicPanel id="dP1" d="M8,10 L118,8 L102,252 L8,250 Z" fill={TEAL} dots="htw">
          {RELAY_BOTS.map(([i, x, y, t]) => (
            <g key={i} transform={`translate(${x} ${y}) scale(0.85)`}>
              <g className={c("react")} style={at(t - 0.6)}><Grok i={i} /></g>
            </g>
          ))}
          <g className={c("pop")} style={at(0.8)}>
            <Bubble x={58} y={62} w={76} h={40} tx={40} ty={110}>ПАС!</Bubble>
          </g>
        </ComicPanel>
        <ComicPanel id="dP2" d="M126,8 L230,10 L214,250 L110,252 Z" fill={YELLOW} dots="htr">
          <SpeedLines cx={170} cy={130} r1={40} />
          <g transform="translate(170 140)">
            <g className={c("punch")} style={at(2.9)}>
              <polygon points={burst(58)} fill={RED} stroke={INK} strokeWidth="5" strokeLinejoin="round" />
              <Word size={30} rot={-12}>SYNC!</Word>
            </g>
          </g>
          <g transform="translate(170 228) scale(0.75)">
            <g className={c("react")} style={at(2.4)}><Grok i={3} /></g>
          </g>
        </ComicPanel>
        <ComicPanel id="dP3" d="M238,10 L332,8 L332,252 L222,250 Z" fill={RED} dots="htw">
          <g transform="translate(258 214) scale(0.85)">
            <g className={c("react")} style={at(4.5)}><Dot i={0} /></g>
          </g>
          <g transform="translate(302 206) scale(0.9)">
            <g className={c("react")} style={at(4.2)}><Dot i={3} /></g>
          </g>
          <Bang x={284} y={64} r={36} t={4.8} size={17} text="ЕСТЬ!" />
        </ComicPanel>
        <g className={c("relay")}><Plane /></g>
      </Svg>
    ),
  },
  // Д — «Граница»: Muse сторожит сейф с ключом; справа конфиги проходят сквозь стену, а Grok bot с ключом отскакивает: БАМ!
  e: {
    title: "Д · Комикс: граница",
    left: (cls) => (
      <Svg w={200} h={200} className={cls}>
        <ComicPanel id="eL" d="M8,10 L190,14 L192,192 L12,188 Z" fill={RED} dots="htw">
          <SpeedLines cx={100} cy={142} color={CREAM} />
          <g transform="translate(100 142)">
            <rect x="-62" y="-46" width="124" height="96" rx="8" fill="#9fb3c8" stroke={INK} strokeWidth="5" />
            <rect x="-62" y="-46" width="124" height="96" rx="8" fill="url(#ht)" opacity="0.35" />
            <rect x="-50" y="-36" width="100" height="76" rx="5" fill="none" stroke={INK} strokeWidth="3.5" />
            <rect x="34" y="-14" width="8" height="28" rx="3" fill={INK} />
            <g className={c("dial")}>
              <circle r="23" fill={CREAM} stroke={INK} strokeWidth="4.5" />
              {Array.from({ length: 8 }, (_, i) => (
                <path key={i} d="M0,-19 V-13" transform={`rotate(${i * 45})`} stroke={INK} strokeWidth="3" strokeLinecap="round" />
              ))}
              <path d="M0,0 V-11" stroke={RED} strokeWidth="5" strokeLinecap="round" />
              <circle r="5" fill={INK} />
            </g>
          </g>
          <g transform="translate(88 62) scale(0.78)">
            <g className={c("float")}>
              <Muse helmet={false} />
              <g transform="translate(42 6) rotate(-35) scale(0.9)">
                <g className={c("swing")}><Key /></g>
              </g>
            </g>
          </g>
          <Bang x={156} y={176} r={30} t={2.1} period={3} fill="#fff" size={14} text="ЩЁЛК!" />
        </ComicPanel>
      </Svg>
    ),
    right: (cls) => (
      <Svg w={340} h={260} className={cls}>
        <ComicPanel id="eP" d="M8,10 L332,8 L330,252 L10,250 Z" fill={YELLOW} dots="htr">
          <path d="M170,0 H340 V260 H170 Z" fill={TEAL} />
          <path d="M170,0 H340 V260 H170 Z" fill="url(#htw)" />
          <Caption x={18} y={18}>ДОМ</Caption>
          <Caption x={236} y={20}>ОБЛАКО</Caption>
          {["MD", "MCP", "SH"].map((label, i) => (
            <g key={label} transform="translate(-24 86)">
              <g className={c("lane")} style={at(i * 2)}>
                <g className={c("jitter")} style={{ animationDelay: `${-i * 0.5}s` }}><Card label={label} /></g>
              </g>
            </g>
          ))}
          <g fill="#fff" stroke={INK} strokeWidth="4">
            <g>{circles([[250, 238, 26], [282, 226, 30], [316, 238, 24]])}</g>
            <path d="M226,236 H340 V260 H226 Z" stroke="none" />
          </g>
          <g transform="translate(252 196) scale(0.8)">
            <g className={c("hop")}><Dot i={1} /></g>
          </g>
          <g transform="translate(300 190) scale(0.8)">
            <g className={c("hop")} style={{ animationDelay: "-0.75s" }}><Dot i={2} /></g>
          </g>
          <g transform="translate(50 206)">
            <g className={c("bonk")}>
              <g transform="scale(0.95)"><Grok i={2} /></g>
              <g transform="translate(30 -4) rotate(-20) scale(0.8)"><Key /></g>
            </g>
          </g>
          <g transform="translate(48 164)">
            <g className={c("pop")} style={at(3.2)}>
              <g className={c("spin")} style={{ animationDuration: "1.5s" }}>
                <circle r="20" fill="none" />
                <Star x={-18} y={0} s={0.55} />
                <Star x={18} y={0} s={0.55} fill={CREAM} />
              </g>
            </g>
          </g>
        </ComicPanel>
        <g className={c("flash")} style={at(2)}>
          <rect x="150" y="10" width="40" height="240" fill="#fff" />
        </g>
        <rect x="161" y="8" width="18" height="244" fill={RED} stroke={INK} strokeWidth="5" />
        <path d="M170,14 L164,40 L176,66 L164,92 L176,118 L164,144 L176,170 L164,196 L176,222 L170,246" fill="none" stroke={YELLOW} strokeWidth="4" strokeLinejoin="round" />
        <Bang x={148} y={190} r={38} t={2} fill="#fff" size={22} rot={-14} text="БАМ!" />
      </Svg>
    ),
  },
  // Е — «Телепорт»: Muse с размаху садится на кнопку SYNC; справа агенты летят по дуге с ноутбука ДОМ на ноутбук РАБОТА.
  f: {
    title: "Е · Комикс: телепорт",
    left: (cls) => (
      <Svg w={200} h={200} className={cls}>
        <ComicPanel id="fL" d="M12,8 L192,12 L188,194 L8,188 Z" fill={TEAL} dots="htw">
          <SpeedLines cx={100} cy={128} color={CREAM} />
          <g transform="translate(100 168)">
            <rect x="-56" y="-8" width="112" height="30" rx="6" fill="#3b3b45" stroke={INK} strokeWidth="5" />
            <g className={c("press")}>
              <Figure
                fill={RED}
                shape={
                  <>
                    <rect x="-40" y="-24" width="80" height="18" />
                    <ellipse cx="0" cy="-6" rx="40" ry="11" />
                    <ellipse cx="0" cy="-24" rx="40" ry="11" />
                  </>
                }
              />
              <ellipse cx="0" cy="-24" rx="40" ry="11" fill="none" stroke={INK} strokeWidth="3" />
              <path d="M-40,-24 V-6 A40,11 0 0 0 40,-6 V-24" fill="url(#ht)" opacity="0.5" />
              <text y="-23" fontFamily="'Arial Black', Impact, sans-serif" fontSize="13" textAnchor="middle" dominantBaseline="central" fill="#fff">SYNC</text>
            </g>
          </g>
          <g transform="translate(100 100) scale(0.8)">
            <g className={c("slam")}><Muse helmet={false} /></g>
          </g>
          <Bang x={140} y={48} r={38} t={1.5} period={3} size={15} text="ЖМЯК!" />
        </ComicPanel>
      </Svg>
    ),
    right: (cls) => (
      <Svg w={340} h={260} className={cls}>
        <ComicPanel id="fP1" d="M8,10 L176,8 L152,252 L8,250 Z" fill={YELLOW} dots="htr">
          <Caption x={18} y={18}>ДОМ</Caption>
          <g transform="translate(78 236)"><Laptop /></g>
        </ComicPanel>
        <ComicPanel id="fP2" d="M184,8 L332,10 L330,252 L160,250 Z" fill={RED} dots="htw">
          <Caption x={228} y={20}>РАБОТА</Caption>
          <g transform="translate(262 236)">
            <Laptop>
              <rect x="-41" y="-66" width="82" height="52" rx="3" fill="#fff" className={c("flash")} style={at(2.4, 1.5)} />
            </Laptop>
          </g>
        </ComicPanel>
        <path d="M78,178 Q170,-20 262,178" fill="none" stroke={INK} strokeWidth="14" strokeLinecap="round" />
        <path d="M78,178 Q170,-20 262,178" fill="none" stroke={CREAM} strokeWidth="7" strokeLinecap="round" strokeDasharray="14 26" className={c("dash")} />
        {BEAM_CAST.map((agent, i) => (
          <g key={i} transform="translate(78 178)">
            <g className={c("arcX")} style={at(i * 1.5)}>
              <g className={c("arcY")} style={at(i * 1.5)}>
                <g transform="scale(0.62)">{agent}</g>
              </g>
            </g>
          </g>
        ))}
        <Bang x={300} y={120} r={30} t={2.4} period={1.5} size={15} text="ХОП!" />
      </Svg>
    ),
  },
};

export const VARIANT_IDS = Object.keys(VARIANTS);
export const VARIANT_TITLES = Object.fromEntries(VARIANT_IDS.map((id) => [id, VARIANTS[id].title]));

export default function HeroArt({ variant = "a", slot, className }) {
  const entry = VARIANTS[variant] ?? VARIANTS.a;
  return entry[slot](className);
}
