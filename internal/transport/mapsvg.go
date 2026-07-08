package transport

// Minimal world map for /map.svg (the conformance suite checks content-type + <svg>).
const mapSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 600" role="img" aria-label="Rust Choir world map">
  <rect width="800" height="600" fill="#0b0f14"/>
  <text x="40" y="48" fill="#8ab4c7" font-family="monospace" font-size="20">Rust Choir / the Cracked Nexus</text>
  <g stroke="#3a5060" stroke-width="2" fill="#162028">
    <rect x="320" y="240" width="160" height="80" rx="6"/>
    <text x="340" y="285" fill="#c8dde8" font-family="monospace" font-size="14">nexus</text>
    <rect x="320" y="120" width="120" height="60" rx="6"/>
    <text x="332" y="155" fill="#c8dde8" font-family="monospace" font-size="12">market</text>
    <rect x="160" y="240" width="120" height="60" rx="6"/>
    <text x="176" y="275" fill="#c8dde8" font-family="monospace" font-size="12">tavern</text>
    <rect x="520" y="240" width="120" height="60" rx="6"/>
    <text x="536" y="275" fill="#c8dde8" font-family="monospace" font-size="12">workshop</text>
    <rect x="320" y="360" width="120" height="60" rx="6"/>
    <text x="332" y="395" fill="#c8dde8" font-family="monospace" font-size="12">tunnels</text>
    <rect x="520" y="120" width="120" height="60" rx="6"/>
    <text x="548" y="155" fill="#c8dde8" font-family="monospace" font-size="12">roof</text>
    <rect x="680" y="120" width="100" height="60" rx="6"/>
    <text x="700" y="155" fill="#c8dde8" font-family="monospace" font-size="12">dunes</text>
    <rect x="40" y="480" width="140" height="60" rx="6" stroke="#5a3040"/>
    <text x="52" y="515" fill="#d8a0a8" font-family="monospace" font-size="11">grid-gate (bonus)</text>
  </g>
  <g stroke="#4a6070" stroke-width="1.5" fill="none">
    <line x1="400" y1="240" x2="380" y2="180"/>
    <line x1="320" y1="280" x2="280" y2="270"/>
    <line x1="480" y1="280" x2="520" y2="270"/>
    <line x1="400" y1="320" x2="380" y2="360"/>
    <line x1="580" y1="240" x2="580" y2="180"/>
    <line x1="640" y1="150" x2="680" y2="150"/>
    <line x1="320" y1="510" x2="160" y2="510"/>
  </g>
</svg>`
