# UI guidelines

Rules for the web UI (`web/`) on desktop and mobile. Routier follows the
[GNOME Human Interface Guidelines](https://developer.gnome.org/hig/) (HIG) and
builds on Cheval UI, whose primitives mirror libadwaita widgets. This page
adapts the HIG to a web router UI and adds the rules specific to Routier. The
frontend section of [code-style.md](code-style.md) covers overlays, forms, and
draft controls. New screens follow both, and existing screens are brought in
line when they are touched.

| HIG / libadwaita | Cheval UI |
|---|---|
| Header bar | `PageHeader` |
| Sidebar (`AdwNavigationSplitView`) | the shell sidebar, `SectionNav` |
| View switcher, view switcher bar | `Tabs`, `Segmented`, `MobileNav`, `MobileSubNav` |
| Boxed list, preferences group | `PreferencesGroup` |
| Action, entry, combo, switch, spin rows | `Row`, `EntryRow`, `ComboRow`, `SwitchRow`, `IntegerEntryRow` |
| Column view | `Table` |
| Status page (placeholder) | `EmptyState` |
| Banner | `NoticeBanner`, layout banners |
| Toast | `toast` (Sonner) |
| Alert dialog | `AlertDialog` |
| Action dialog, secondary window | `Sheet` (editing), `Dialog` (read-only) |
| Spinner | `Spinner` |

## Principles

### Audience

The HIG asks for the vocabulary of the people using the app. Routier's users
are network and system administrators, so their vocabulary is the protocol and
kernel vocabulary.

* Use the real terms as is: `BGP`, `OSPF`, `VRRP`, `prefix list`, `route map`,
  `masquerade`. Do not expand acronyms or rename concepts into consumer language.
* Do not teach protocols. A screen configures BGP; it does not explain what BGP is.
* Show values the way the operator types them: addresses with their prefix
  length, interface names, ASNs, file paths. Machine values use `font-mono`.
* Expose the underlying setting rather than hiding it behind an abstraction.
  Sensible defaults are welcome; hidden behavior is not.

### Make it simple

* One way to do each thing. The same kind of entity is created, edited,
  inspected, and deleted the same way on every page.
* The structure follows the configuration model: a page per area, a group per
  section, a row per setting.
* Each view has one clear subject. Do not pack unrelated content into a view.
* Frequent actions are close at hand; rare and dangerous ones are further away
  (a menu, the end of a form), never next to the primary action.

### Reduce effort

* Do automatically what can be done automatically: derive values, prefill
  defaults, suggest known interfaces, addresses, and variables.
* Minimize steps. Do not add a tab, a wizard step, or a confirmation that does
  not prevent a real mistake.
* Make invalid actions insensitive (disabled) instead of reporting an error
  after the click.

### The UI explains itself

The layout, labels, grouping, placeholders, and defaults carry the meaning. Text
that explains the interface is a sign the interface needs restructuring.

* No introductory paragraphs on pages, tabs, or cards.
* `PageHeader` carries a title and actions. A description is allowed only for
  live context the title cannot give (the instance being configured, a refresh
  interval), never a summary or list of what the page contains.
* Row and group descriptions state facts the operator cannot infer and that
  change a decision: a valid range, a non-obvious side effect (restarts a
  daemon, recreates a bond), where the value is written, or data that may be
  overwritten. One short sentence. Omit it when the title is enough.
* Text is always justified for consequences that are otherwise invisible:
  irreversible effects, connectivity rollback, and secrets shown only once.

## Layout and navigation

### Shell

* The sidebar lists the top-level areas, in groups, ordered by frequency of use.
  Each item has a symbolic icon and a label.
* App-wide commands (Preferences, appearance, Sign Out, About Routier) live in
  the main menu, a single menu button at the top of the sidebar, not as a row of
  buttons in the sidebar footer. On mobile they live on the More page.
* On mobile, the bottom bar holds the top-level groups. Each group tab opens a
  list page of its items (icon rows with a chevron); tapping the tab again
  returns to that list. No pill strips or horizontally scrolling sub-menus.
* Persistent states use banners under the header, one line each: pending draft,
  apply failure, remote instance, announcements.

### Pages

* Every routed page renders exactly one `PageHeader`, first on the page, whose
  title matches its navigation label. Nested panels and sections never render
  their own page header.
* `PageHeader` actions are the header bar: few controls (up to three), the
  primary action last, rarely used or destructive actions in a menu button.
  Header actions may combine an icon and a label and use the ghost style.
* Preference content (forms, boxed lists) sits in a centered column with a
  maximum width (`max-w-3xl`), so rows do not stretch across wide screens.
  Tables, charts, and graphs may use the full width.

### Hierarchy

* At most one level below a sidebar item: either view switcher tabs
  (`Tabs`, three to five views, nouns), or a section list (`SectionNav`) when
  there are more. Do not nest tabs inside tabs or section lists inside tabs.
* Rows that open another view or detail page end with a chevron
  (`ChevronRight`) and open on click anywhere on the row.
* The back button sits at the top left (mobile header) and returns to the parent.
  It is the only back control on mobile: no in-page back links or back rows there.
* A group title never repeats the section or page title above it.
* Overlays never stack: a sheet or dialog does not open another sheet or dialog.

## Containers

### Boxed lists (`PreferencesGroup`)

* Small, static sets of settings and entities go in boxed lists. Large or live
  data (leases, routes, logs, neighbors) goes in a `Table`.
* Several groups on one page act as sections, each with a short heading.
* A row holds one control, two at most. Clicking the row background operates
  its control.
* Editable lists: an Add button in the group header (or an add row at the end),
  and the remove button at the end of each row. Reordering uses drag handles at
  the start of the row plus Move Up / Move Down in the row menu.
* Multiple text elements in a row are told apart by weight and color (title,
  muted subtitle), not by extra sizes.

### Tables (`Table`)

* Rows are simple and regular, one or two elements per cell.
* Default order is the most useful one (interfaces by name, leases by expiry,
  logs newest first). Sortable columns show a down arrow for natural order and
  an up arrow for reverse.
* The identifying column (name, address) comes first and stays visible on
  small screens.
* On phones, a table with a header renders each row as a card: the first cell
  (or the cell marked `data-title`) is the card title, the others are
  label/value lines. Long free text (command lines, messages) is marked
  `data-wide` and shows under its label, full width and left-aligned. Tables
  without a header (key/value lists) stay tables; `cards={false}` opts out.

### Sheets and dialogs

* `Sheet` edits or creates one entity; `Dialog` shows read-only information or
  runs an action with options; `AlertDialog` confirms (see code-style.md for the
  per-feature choice).
* Overlays open only in response to a user action, never unexpectedly.
* The heading names the action or entity ("Add Neighbor", "Interface eth0").
* Buttons: Cancel first, the affirmative action last on the right, labelled
  with a specific verb ("Save", "Delete Zone"), never "OK" or "Done".
* Esc cancels. Return triggers the affirmative button unless it is destructive.
  Initial focus goes to the first control the user is expected to use.
* The affirmative button stays insensitive until required fields are valid.

### Menus and popovers

* Menu items are verbs for commands and adjectives for settings, three to
  twelve items, grouped by kind, no nested submenus.
* Popovers are small (under a third of the view) and close on Esc and on outside
  click; they have no Close or Done button.

## Controls

### Buttons

* Labels are short imperative verbs in header capitalization: "Add Route",
  "Apply", "Save Draft".
* A button carries a label or an icon, not both, except in the page header and
  the layout banners. Icon-only buttons use widely recognized symbols (add,
  remove, edit, copy, refresh, menu, back) and always have a tooltip and an
  `aria-label`.
* One emphasized button per view: either the suggested style for the next step
  or the destructive style for a dangerous one, never several.
* Buttons placed together share the same height and style.
* No actions on double click or right click only.

### Switches, checkboxes, radios, drop-downs

* Binary settings use a switch (`SwitchRow`). Checkboxes are for selecting items
  in a list and for options inside a form that are not settings.
* Labels state the positive: "Accept Router Advertisements", not
  "Disable RA".
* A choice of two options uses `Segmented` or a switch; three to five short
  options use `Segmented`; longer lists use a drop-down (`ComboRow`, `Select`).
  Never a drop-down with fewer than three items.
* A control does not change the value of another; it may change its
  sensitivity or visibility. A switch that reveals dependent rows sits
  directly above them.

### Text fields

* Every field has a row title. The placeholder shows an example value
  (`192.0.2.1/24`, `65001`), or, for a search field, an instruction ending in an
  ellipsis ("Search Leases…").
* Field width follows the expected content (an MTU field is narrow, a
  description is wide) inside the shared control column.
* Validate in real time while typing, inline under the field, and prefer a
  quiet positive state over shouting while the user is still typing.
* Passwords and keys use a password field with a reveal button.
* Fields that accept known values (interfaces, variables, prefix lists, sysctl
  keys) offer completion with `AutocompleteInput`, fed by the live values when
  the host has them.
* Values that can be long (TXT records, DUIDs, IPv6 addresses, hostnames) wrap
  instead of truncating: use `AutoTextarea` (single line, grows with content).

## Feedback

* Toasts report the result of an action the user just took, with a short title
  and at most one button. They appear at the bottom center.
* Destructive actions inside a draft (removing a row, a neighbor, a rule) need no
  confirmation: Discard Edits undoes them. Destructive actions with immediate
  effect (reboot, delete an API key, wipe, unpair a friend) use `AlertDialog`,
  or a toast with Undo when the action can be reverted.
* Banners describe ongoing states in one line, with at most one button
  ("Configuring rt2", "Apply failed"). No explanations.
* A view that failed to load shows the error and a Retry action, never an
  endless spinner.
* Spinners appear only for operations that may take more than about a second,
  close to the element they relate to, one at a time. Operations over about 30
  seconds (apply, upgrade, backup) show progress or a streamed log instead.
* Empty views use `EmptyState`: a symbolic icon, a heading in header
  capitalization ("No Neighbors"), optionally one line, and the action to add
  content. Empty search results say "No Results".
* Tooltips are short, header capitalized, do not repeat the visible label, and
  never carry essential information. If one control in a group has a tooltip,
  all do.

## Writing

### Capitalization

* Header capitalization for page titles, group headings, tabs, sidebar items,
  buttons, menu items, row titles, tooltips, and empty-state headings:
  capitalize words of four letters or more, verbs, nouns, and the first and last
  word ("Add a Route", "Hold Time", "Return to This Instance").
* Sentence capitalization for descriptions, banner and toast bodies, checkbox
  labels inside forms, placeholders, and any text that forms a sentence.
* Acronyms and protocol keywords keep their own case (`BGP`, `nftables`,
  `dhcpcd`, `IPv6`).

### Style

* Short and neutral. Avoid "you" and "my"; use "your" only when ownership must
  be stated.
* No trailing period on titles, labels, descriptions, or single-sentence text.
* Ellipsis (…) at the end of a command that needs more input before it acts:
  a button that opens a sheet or a confirmation ("Add Neighbor…",
  "Delete…"). In-progress labels also use it ("Applying…").
* Headings omit articles and auxiliary verbs ("Apply Failed", not
  "The Apply Has Failed").
* Write "for example", not "e.g." or "i.e.".
* Use the same term for the same thing everywhere, matching the CLI and docs.
* Typography characters: `…` (not `...`), `×` for dimensions, `’` for
  apostrophes in prose, `“ ”` for quotes, an en dash for ranges, and a space
  between a number and its unit (`32 GB`, `100 ms`). Pluralize properly
  ("1 neighbor", "2 neighbors"), never "neighbor(s)".
* No em dashes.

### Typography

* A small fixed set of styles: page title (`PageHeader`), heading
  (`text-sm font-semibold`), body (`text-sm`), caption (`text-xs`), and caption
  heading (`text-xs font-semibold`). Large numbers in stat cards use the
  `StatCard` style.
* No arbitrary sizes (`text-[10px]`, `text-[11px]`, `text-[13px]`). A denser
  size that is genuinely needed lives in a primitive.
* No italics and no all-caps text. Hierarchy comes from weight and color.

## Style

* Colors come from semantic tokens only: `success`, `warning`, `danger`, `info`,
  `muted`, `primary`. No Tailwind palette colors (`red-500`, `emerald-400`) in
  feature code.
* `danger` expresses state (down, failed, error). `destructive` is the button
  style for an action that removes or stops something.
* Live state uses `StatusChip` / `StateChip` with `intentForState`, so `up`,
  `MASTER`, `Established`, and `failed` look the same everywhere. `Badge` is for
  static labels (type, family, tag).
* Color never carries meaning alone: the state is also written or shaped.
* No blinking or pulsing elements to signal state.
* Icons are the symbolic (outline, monochrome) Lucide set at one size per
  context: `h-4 w-4` in rows and buttons, `h-3.5 w-3.5` in the sidebar.
* Appearance offers Light, Dark, and Follow System; the default follows the
  system.

## Input

### Pointer and touch

* Never rely on hover to reveal actions or information: row actions, copy
  buttons, and menus are visible or reachable by tap.
* Click and touch targets are at least 32 px (the default `Button` height);
  navigation controls on mobile (back, tabs) are at least 40 px.
* Do not reference input devices in text ("click", "hover"); name the control.
* On touch screens, opening a sheet or dialog never focuses a field, so the
  keyboard stays closed until the user taps one (Cheval UI ignores `autoFocus`
  on coarse pointers).
* iOS gets `maximum-scale=1` so focusing a field does not zoom the page; pinch
  zoom still works.
* Popups anchored to fields position against the visual viewport, so they stay
  above the on-screen keyboard.
* A card that opens a detail view is clickable as a whole, with a chevron;
  its own action buttons stay separate.

### Keyboard

* Everything is reachable and operable with the keyboard, in a logical focus
  order, with a visible focus ring.
* Esc closes the topmost transient element (menu, popover, dialog, sheet).
* Standard shortcuts where the function exists: Ctrl+F search, Ctrl+S save draft,
  Ctrl+, preferences, Alt+← back, Ctrl+R reload the current view.
* Search fields filter live and react to typing in the list they filter.

## Adaptive layout

The same UI serves phones, designed from the smallest size up. The HIG minimum
for phone-capable apps is 360×294 px.

* Every screen works at 360 px wide with no horizontal page scroll.
* Multi-column grids collapse to one column below `sm`: write
  `grid-cols-1 sm:grid-cols-2`, never a bare `grid-cols-2` around fields.
* Preference rows use the shared 20rem control column on desktop and stack
  title above control on mobile. Switches always stay at the row end.
* Tables render as cards on phones (see Tables).
* Section lists become a boxed list on narrow screens; opening a section is a
  history step (the header back button returns to the list), the page title
  shows the section title (`title` on the item when the label alone is
  ambiguous, e.g. BGP for BGP > General), and a URL naming a section opens it
  directly. Tab strips
  do not overflow off-screen.
* Editable lists of entities (DNS records, DHCP reservations) show each entity
  as a card on phones, with labelled fields and every value fully visible.
* Wide graphics (switch faceplate) scale down to a minimum and then scroll on
  their own; the page itself never scrolls sideways.
* Banners fit one line on phones: a short title, the key value (a countdown),
  and one button.
* Sheets and dialogs use the Cheval UI components, which go full height and
  respect safe areas on mobile. Do not size overlays with fixed widths.
* The primary action stays visible on mobile; secondary actions may collapse
  into a menu.

## Accessibility

* Every control has an accessible name; icon-only buttons have `aria-label`.
* Labels precede their controls in focus order and are associated with them.
* Check new screens in light and dark, with the browser zoomed to 200 %, and
  with keyboard only.
