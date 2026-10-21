import styles from "./hero-sticker.module.css";

// Комикс «Телепорт». Все анимации — CSS-циклы: кадр 0% равен кадру 100%, периоды делят 6 с.
const INK = "#1b1b1f";
const YELLOW = "#ffc61a";
const RED = "#e8452c";
const TEAL = "#2ba6b5";
const CREAM = "#f4ecd2";

const c = (...names) => names.map((name) => styles[name]).join(" ");

// Сдвиг анимации так, чтобы её кадр 0% пришёлся на секунду t цикла period.
const at = (t, period = 6) => ({ animationDelay: `${(t % period) - period}s`, animationDuration: `${period}s` });

const round = (n) => Math.round(n * 10) / 10;

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
    fill: "#1287ff",
    shape: <path d="M0,-26 Q7,-26 13,-15 L27,10 Q33,24 15,24 L-15,24 Q-33,24 -27,10 L-13,-15 Q-7,-26 0,-26 Z" />,
    eyes: [[-6, 11, 5, 11, 0], [6, 11, 5, 11, 0]],
  },
  {
    fill: "#ff6a00",
    shape: <>{circles([[-11, -8, 14], [6, -11, 14], [15, 5, 13], [3, 12, 12], [-14, 8, 12]])}</>,
    eyes: [[-5, -10, 5, 14, -70], [9, -5, 5, 14, -65]],
  },
];

function Grok({ i }) {
  const bot = GROK[i];
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
  const dot = DOTS[i];
  return <Figure fill={dot.fill} shape={dot.shape} fluffy>{dot.face}</Figure>;
}

const MUSE_BODY = "M-32,40 Q-37,4 -32,-22 Q-25,-48 0,-48 Q25,-48 32,-22 Q37,4 32,40 Q0,47 -32,40 Z";

// Meta Muse: кремовый пушистый капюшон с лицом, центр в (0,0).
function Muse() {
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
    </g>
  );
}

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

// Звукоподражание во вспышке: появляется в секунду t и гаснет.
function Bang({ x, y, r, t, period = 6, size, text }) {
  return (
    <g transform={`translate(${x} ${y})`}>
      <g className={c("pop")} style={at(t - 0.1, period)}>
        <polygon points={burst(r)} fill={YELLOW} stroke={INK} strokeWidth="4" strokeLinejoin="round" />
        <text
          transform="rotate(-8)"
          fontFamily="'Arial Black', Impact, sans-serif"
          fontStyle="italic"
          fontWeight="900"
          fontSize={size}
          textAnchor="middle"
          dominantBaseline="central"
          fill={RED}
          stroke={INK}
          strokeWidth={size / 6}
          strokeLinejoin="round"
          paintOrder="stroke"
        >
          {text}
        </text>
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

// Ноутбук, центр основания в (0,0); children рисуются поверх экрана.
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

function Svg({ w, h, className, children }) {
  return (
    <svg className={className} viewBox={`0 0 ${w} ${h}`} aria-hidden="true" focusable="false">
      <Defs />
      <g filter="url(#sticker)">{children}</g>
    </svg>
  );
}

const BEAM_CAST = [<Grok key="g0" i={0} />, <Dot key="d1" i={1} />, <Grok key="g1" i={1} />, <Dot key="d0" i={0} />];

const SLOTS = {
  // Muse с размаху садится на кнопку SYNC: ЖМЯК!
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
          <g className={c("slam")}><Muse /></g>
        </g>
        <Bang x={140} y={48} r={38} t={1.5} period={3} size={15} text="ЖМЯК!" />
      </ComicPanel>
    </Svg>
  ),
  // Агенты по одному летят по дуге с ноутбука ДОМ на ноутбук РАБОТА.
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
};

export default function HeroArt({ slot, className }) {
  return SLOTS[slot](className);
}
