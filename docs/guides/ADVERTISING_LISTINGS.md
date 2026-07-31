# DOGMud — Listing-Site Registration Kit (2026-07-28)

Everything needed to register Delusions of Grandeur on the MUD directories.
Copy is ready to paste; tweak voice freely. MSSP is live and externally
validated, so crawler-based sites will pick up player counts automatically.

## Connection details (used by every site)

- **Name:** Delusions of Grandeur (DOGMud)
- **Host / port:** `dogmud.org` : `33333` (telnet)
- **Play in browser:** https://www.dogmud.org
- **Codebase:** GoMud (heavily customized fork)
- **Status:** Open Beta · Free — no pay-to-play, no pay-for-perks
- **MSSP:** enabled on the telnet port (name, live players, world counts)

## The sites

**Status 2026-07-28: ✅ TMC registered · ✅ MudStats registered · ✗ Top Mud
Sites (stopped accepting new votes — skipped) · ✗ r/MUD (owner's account has
negative karma there from a prior attempt — do NOT post from it; if ever,
an organic post from a PLAYER's established account is the only viable
route) · ✗ Grapevine (their registration mailer is dead).** Remaining
low-effort channels if wanted: MUD community Discords (announcements
channels), MUD Coders Guild.

| Site | URL | Notes |
|------|-----|-------|
| Grapevine | https://grapevine.haus/register/new | MUD listing + cross-game chat network; polls games for live status. **⚠ Effectively unmaintained (repo dormant since 2021) and its registration verification email appears broken (tried 2026-07-28, never arrived).** If wanted: retry with a different email provider, else contact the maintainer via https://grapevine.haus/contact or a GitHub issue (oestrich/grapevine). Its public MSSP checker (https://grapevine.haus/mssp) works WITHOUT an account. Not worth blocking the announcement on. |
| The MUD Connector (TMC) | https://www.mudconnect.com | The old standby directory. Wants the long description + feature list. |
| MudStats | https://mudstats.com | Submit host+port; it crawls MSSP on its own from there. Minimal copy needed. |
| Top Mud Sites | https://www.topmudsites.com | Vote-ranked directory; medium description. |
| r/MUD | https://reddit.com/r/MUD | See the cautions below — a previous post was removed. |

## Logo / banner assets

Generated 2026-07-28 (chrysalis + three moons emblem, matches the web
client's leather aesthetic), in
`_datafiles/html/public/static/images/branding/`:

- `emblem_1024.png` — master square emblem (downscale to any size a site asks for)
- `logo_256.png` / `logo_100.png` — ready-made small squares
- `banner_468x60.png` — classic listing banner (emblem + title + dogmud.org)

Once deployed they are also web-served, e.g.
`https://www.dogmud.org/static/images/branding/banner_468x60.png` for sites
that want a hosted image URL.

## Short blurb (~45 words — Grapevine, MudStats)

> On the colony world of Gaius, conviction reshapes flesh. No levels, no
> classes — you become what you practice, and the Chrysalis grows you
> mutations to match. A living economy, NPCs with daily lives, and a world
> with something older buried beneath it. Free, open beta, playable in the
> browser.

## Long description (~180 words — TMC, Top Mud Sites)

> Delusions of Grandeur is set on Gaius, a world of walled towns, trade
> roads, and wild country where belief changes the world: a symbiotic
> organism in every living thing makes conviction physically real. Magic is faith made
> manifest, and the way you actually play — blade, spellcraft, or sheer
> force of voice — slowly drifts your body into mutations that fit it.
>
> There are no levels and no experience points. Every stat and skill grows
> only through use. Combat resolves across three channels — body, belief,
> and voice — and every path is viable from your first fight.
>
> The world runs whether you're there or not: shopkeepers price by scarcity,
> NPC buyers bid at the auction house, caravans and ferries move goods,
> townsfolk keep daily schedules, gossip, and remember what they've seen.
> Three moons pull at the magic. Seasons and weather roll through. And under
> the oldest hills, something that fell from the sky a very long time ago is
> still waiting to be understood.
>
> Free to play, open beta. Telnet at dogmud.org:33333 or play in the
> browser at dogmud.org — new players get a guided tutorial; veterans of
> the genre can skip straight in.

## Feature bullets (TMC "features" box)

- Use-based progression only — no levels, no XP, no classes
- Mutation system: your playstyle drifts your body toward matching powers
- Three combat channels (physical / magical / conviction) with full parity
- Living economy: dynamic shop pricing, player+NPC auction house, caravans,
  ferries, warehouse storage, salvage and crafting chains
- NPCs with schedules, patrols, relationships, and NPC-to-NPC conversation
- Weather, seasons, three moons with mechanical pull, day/night
- Guilds (social), achievements, global chat/newbie/trade channels
- Browser client with a live-drawn zone map, vitals, quest tracker
- Guided tutorial for MUD newcomers; fast lane for veterans
- Seamless live restarts — updates land without dropping your connection
- 1,300+ rooms across 49 zones; free, no monetization

## r/MUD post — draft + cautions

**Cautions first (a previous post was removed):**

1. Read the current sidebar rules before posting — promo norms shift, and
   some periods route all advertising into a pinned monthly thread. If a
   promo/advertising flair exists, USE it.
2. Post from the account with the most age/history available, and stick
   around to answer comments the same day — drive-by ads are what get
   nuked. Reddit-wide 90/10 self-promo norms apply.
3. Frame it as a dev/launch story with substance, not an ad. The draft
   below is written that way.
4. If it's removed again: message the mods and ask what venue they prefer
   rather than reposting. Alternatives with friendlier promo norms: the
   MUD Coders Guild community, MUD Discord servers, and Grapevine's own
   visibility.

**Title:** After two years of building, our no-levels MUD is in open beta —
mutations that follow your playstyle, and a world that runs without you

**Body:**

> I've been building Delusions of Grandeur (DOGMud) on a customized GoMud
> fork, and it's finally at the point where strangers can break it — so:
> open beta.
>
> The design bet: no levels, no XP, no classes. Everything advances by use,
> and the world's central organism — the Chrysalis — watches *how* you
> fight and live, then grows you mutations that fit. Play a grappler long
> enough and your body answers. The same system drives the setting: belief
> is physically real on Gaius, magic is conviction made manifest, and the
> "voice" path (rhetoric, taunts, presence) is a full combat channel with
> the same weight as blade or spell.
>
> The other obsession has been aliveness: shopkeepers reprice on scarcity
> and their own gold, NPC buyers bid against you at auction, caravans and
> ferries actually haul the goods, townsfolk keep daily schedules and talk
> to each other about you. Weather fronts, seasons, three moons with real
> mechanical pull.
>
> It's free, no monetization, telnet at dogmud.org:33333 or in the browser
> at dogmud.org. Complete-newcomer tutorial if you've never MUDded; skip
> lane if you have. I'd genuinely value blunt feedback from this crowd —
> the last playtest rounds reshaped half the onboarding.

## Post-registration checklist

- [ ] Optionally set `HOSTNAME` / `PORT` / `CONTACT` in the droplet's
      `config-production.yaml` MSSP block (crawler-facing completeness)
- [ ] Run https://grapevine.haus/mssp against dogmud.org:33333 after
      registering there
- [ ] Confirm MudStats picked up the crawl within a day or two
- [ ] Watch `PLAYERS` in MSSP responses — listing sites display it live
