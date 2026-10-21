import styles from "./hero-sticker.module.css";

// Все анимации — CSS-циклы: кадр 0% равен кадру 100%, период кратен 6 с.
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

// Ракета с космонавтом, нос вправо, центр в (0,0).
function Rocket() {
  return (
    <g>
      <g transform="translate(-86 0)">
        <g className={c("flame")}>
          <path d="M0,0 L-52,-20 L-34,-7 L-66,0 L-34,7 L-52,20 Z" fill={RED} stroke={INK} strokeWidth="4" strokeLinejoin="round" />
          <path d="M-4,0 L-30,-9 L-20,0 L-30,9 Z" fill={YELLOW} />
        </g>
      </g>
      <path d="M-62,-18 L-96,-52 L-30,-26 Z" fill={RED} stroke={INK} strokeWidth="5" strokeLinejoin="round" />
      <path d="M-62,18 L-96,52 L-30,26 Z" fill={RED} stroke={INK} strokeWidth="5" strokeLinejoin="round" />
      <path d="M-84,-22 L20,-28 Q75,-20 102,0 Q75,20 20,28 L-84,22 Z" fill={YELLOW} stroke={INK} strokeWidth="5" strokeLinejoin="round" />
      <path d="M-84,10 L20,16 Q60,12 90,6 Q75,20 20,28 L-84,22 Z" fill="url(#ht)" />
      <path d="M58,-22 Q88,-14 102,0 Q88,14 58,22 Q66,0 58,-22 Z" fill={RED} stroke={INK} strokeWidth="5" strokeLinejoin="round" />
      <rect x="-58" y="-26" width="14" height="52" fill={RED} stroke={INK} strokeWidth="4" />
      <circle cx="18" cy="4" r="11" fill={TEAL} stroke={INK} strokeWidth="4" />
      <circle cx="18" cy="8" r="6" fill="url(#ht)" />
      <g transform="translate(-14 -24)">
        <g className={c("astro")}>
          <path d="M-14,0 L-14,-30 Q-14,-38 -6,-38 L10,-38 Q18,-38 18,-30 L18,0 Z" fill={CREAM} stroke={INK} strokeWidth="4" strokeLinejoin="round" />
          <path d="M-14,-14 L18,-14 L18,0 L-14,0 Z" fill="url(#ht)" opacity="0.7" />
          <g transform="translate(14 -30)">
            <g className={c("flag")}>
              <path d="M0,0 L26,-38" stroke={INK} strokeWidth="11" strokeLinecap="round" />
              <path d="M0,0 L26,-38" stroke={CREAM} strokeWidth="5" strokeLinecap="round" />
              <path d="M26,-38 V-64" stroke={INK} strokeWidth="5" strokeLinecap="round" />
              <g className={c("wave")}>
                <path d="M28,-64 L62,-64 L62,-38 L28,-38 Z" fill={RED} stroke={INK} strokeWidth="4" strokeLinejoin="round" />
                <path d="M28,-55 L62,-55 L62,-47 L28,-47 Z" fill={CREAM} />
                <path d="M28,-64 L42,-64 L42,-38 L28,-38 Z" fill={TEAL} stroke={INK} strokeWidth="3" />
              </g>
            </g>
          </g>
          <circle cx="2" cy="-52" r="25" fill={TEAL} stroke={INK} strokeWidth="5" />
          <circle cx="2" cy="-48" r="22" fill="url(#ht)" />
          <path d="M-12,-62 Q-6,-72 6,-70" fill="none" stroke={CREAM} strokeWidth="5" strokeLinecap="round" />
          <rect x="-16" y="-34" width="36" height="8" rx="3" fill={YELLOW} stroke={INK} strokeWidth="3.5" />
        </g>
      </g>
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

const VARIANTS = {
  // А — «Лунный старт»: луна слева, ракета с нотами справа.
  a: {
    title: "А · Лунный старт",
    left: (cls) => (
      <Svg w={200} h={200} className={cls}>
        <g className={c("float")}>
          <g className={c("sway")}><Moon cx={100} cy={104} r={78} id="mA" /></g>
        </g>
        <Star x={28} y={28} s={1} delay={0} />
        <Star x={176} y={44} s={0.7} delay={0.5} fill={TEAL} />
        <Star x={168} y={176} s={0.9} delay={1} fill={RED} />
      </Svg>
    ),
    right: (cls) => (
      <Svg w={340} h={260} className={cls}>
        <g className={c("float")} style={{ animationDuration: "6s" }}>
          <Moon cx={236} cy={146} r={86} id="mA2" />
        </g>
        <g className={c("trail")}>
          <path d="M20,150 H64" stroke={INK} strokeWidth="9" strokeLinecap="round" strokeDasharray="22 22" />
          <path d="M10,184 H60" stroke={INK} strokeWidth="9" strokeLinecap="round" strokeDasharray="22 22" />
        </g>
        <Note x={70} y={120} delay={0} />
        <Note x={44} y={110} delay={1.5} fill={TEAL} />
        <Star x={304} y={30} delay={0.3} />
        <g transform="translate(170 150) rotate(-26) scale(0.92)">
          <g className={c("fly")}><Rocket /></g>
        </g>
      </Svg>
    ),
  },
  // Б — «Крейсер»: ракета неподвижна в кадре, мимо плывёт параллакс из звёзд и луны.
  b: {
    title: "Б · Крейсер",
    left: (cls) => (
      <Svg w={200} h={200} className={cls}>
        <g transform="translate(100 104)">
          <g className={c("spin")} style={{ animationDuration: "12s" }}>
            <ellipse cx="0" cy="0" rx="92" ry="26" fill="none" stroke={INK} strokeWidth="14" transform="rotate(-22)" />
            <ellipse cx="0" cy="0" rx="92" ry="26" fill="none" stroke={YELLOW} strokeWidth="7" transform="rotate(-22)" />
          </g>
          <circle r="50" fill={TEAL} stroke={INK} strokeWidth="5" />
          <circle r="50" fill="url(#ht)" clipPath="none" opacity="0.5" />
          <path d="M-40,-14 Q0,-28 40,-14" fill="none" stroke={CREAM} strokeWidth="6" strokeLinecap="round" />
          <path d="M-46,18 Q0,36 46,16" fill="none" stroke={RED} strokeWidth="6" strokeLinecap="round" />
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
            {[[40, 46], [120, 210], [200, 60], [270, 190], [310, 90]].map(([x, y], i) => (
              <circle key={i} cx={x} cy={y} r="3.5" fill={CREAM} />
            ))}
          </Scroll>
          <Scroll w={340} speed="6s">
            <Moon cx={170} cy={196} r={62} id="mB" />
            <Star x={60} y={70} delay={0.4} />
            <Star x={270} y={54} s={0.8} delay={1} fill={TEAL} />
          </Scroll>
          <g className={c("trail")} style={{ animationDuration: "0.4s" }}>
            <path d="M-10,116 H70" stroke={CREAM} strokeWidth="6" strokeLinecap="round" strokeDasharray="26 40" />
            <path d="M-10,152 H60" stroke={CREAM} strokeWidth="6" strokeLinecap="round" strokeDasharray="26 40" />
          </g>
          <g transform="translate(190 120) rotate(-8) scale(0.9)">
            <g className={c("jitter")}><Rocket /></g>
          </g>
        </Panel>
      </Svg>
    ),
  },
  // В — «Бит»: вспышка за ракетой, прыгающая надпись SYNC!, микрофон слева. Период бита 1,5 с.
  c: {
    title: "В · Бит",
    left: (cls) => (
      <Svg w={200} h={200} className={cls}>
        <g transform="translate(96 108)">
          <g className={c("swing")}>
            <rect x="-9" y="10" width="18" height="70" rx="9" fill={RED} stroke={INK} strokeWidth="5" />
            <rect x="-11" y="26" width="22" height="9" fill={YELLOW} stroke={INK} strokeWidth="3.5" />
            <circle cx="0" cy="-22" r="34" fill={TEAL} stroke={INK} strokeWidth="5" />
            <circle cx="0" cy="-16" r="30" fill="url(#ht)" />
            <path d="M-20,-38 Q-10,-54 8,-52" fill="none" stroke={CREAM} strokeWidth="6" strokeLinecap="round" />
            <path d="M-30,-22 H30 M0,-54 V10" stroke={INK} strokeWidth="3" opacity="0.5" />
          </g>
        </g>
        <Note x={50} y={90} delay={0} />
        <Note x={168} y={70} delay={1.5} fill={TEAL} />
        <Star x={170} y={170} delay={0.6} fill={RED} />
      </Svg>
    ),
    right: (cls) => (
      <Svg w={340} h={260} className={cls}>
        <g transform="translate(170 150)">
          <g className={c("spin")} style={{ animationDuration: "12s" }}>
            {Array.from({ length: 12 }, (_, i) => (
              <path key={i} d="M0,0 L-20,-92 L20,-92 Z" transform={`rotate(${i * 30})`} fill={i % 2 ? YELLOW : TEAL} stroke={INK} strokeWidth="4" strokeLinejoin="round" />
            ))}
          </g>
          <circle r="96" fill="url(#ht)" opacity="0.35" />
        </g>
        <g transform="translate(170 150) rotate(-24) scale(0.82)">
          <g className={c("beat")}><Rocket /></g>
        </g>
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
        <Star x={26} y={96} delay={0.2} />
        <Star x={316} y={214} s={0.9} delay={0.9} fill={RED} />
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
