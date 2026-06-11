import {
  BarChart,
  Button,
  Card,
  CardBody,
  CardHeader,
  Checkbox,
  CollapsibleSection,
  computeDAGLayout,
  Grid,
  H1,
  PieChart,
  Row,
  Select,
  Spacer,
  Stack,
  Stat,
  Text,
  TextInput,
  useCanvasState,
  useHostTheme,
} from "cursor/canvas";

type Edge = [number, string];
type AppEntry = [string, string, string];
type Product = { n: string; c: string; a: string; e: Edge[]; p?: AppEntry[]; pol?: string[] };
type Shared = { b: string; n: number; p: string[] };
type TopologyData = {
  m: {
    admin_url: string;
    exported_at: string;
    product_count: number;
    backend_count: number;
    application_count: number;
  };
  cat: Record<string, string>;
  catCounts: Record<string, number>;
  backends: string[];
  products: Product[];
  shared: Shared[];
};

const DATA = {"m":{"admin_url":"https://tenant-admin.example.com","application_count":2,"backend_count":2,"exported_at":"2026-06-05T12:00:00Z","include_applications":true,"incomplete":false,"product_count":2,"schema_version":"1.0"},"cat":{"B":"Business API","I":"Integration (-IS)","P":"Platform / misc","S":"SAP"},"catCounts":{"B":2,"I":0,"P":0,"S":0},"backends":["billing_api","shared_payments"],"products":[{"n":"seed_alpha","c":"B","a":"API Key","e":[[1,"/payments"]],"p":[["Alpha App","Basic","live"]],"pol":["cors"]},{"n":"seed_multi_backend","c":"B","a":"OIDC","e":[[1,"/payments"],[0,"/billing"],[0,"/invoices"]],"p":[["Multi App","Default","live"]],"pol":["edge_limit","url_rewriting"]}],"shared":[{"b":"shared_payments","n":2,"p":["seed_alpha","seed_multi_backend"]}]} as TopologyData;

const PAGE_SIZE_OPTIONS = [
  { value: "10", label: "10 / page" },
  { value: "20", label: "20 / page" },
  { value: "50", label: "50 / page" },
  { value: "100", label: "100 / page" },
];

type TableSortKey =
  | "product"
  | "category"
  | "auth"
  | "backends"
  | "apps"
  | "policies"
  | "policyNames";
type TableSortDir = "asc" | "desc";

const TABLE_COLUMNS: { key: TableSortKey; label: string; numeric?: boolean }[] = [
  { key: "product", label: "Product" },
  { key: "category", label: "Category" },
  { key: "auth", label: "Auth" },
  { key: "backends", label: "Backends", numeric: true },
  { key: "apps", label: "Apps", numeric: true },
  { key: "policies", label: "Policies", numeric: true },
  { key: "policyNames", label: "Policy names" },
];

function formatPolicyChain(names?: string[]): string {
  if (!names || names.length === 0) {
    return "—";
  }
  return names.join(" → ");
}

function columnFlex(key: TableSortKey): number {
  switch (key) {
    case "product":
      return 2;
    case "category":
    case "auth":
      return 1;
    case "backends":
    case "apps":
    case "policies":
      return 0.65;
    case "policyNames":
      return 3.5;
  }
}

function ProductDataTable({
  products,
  sortKey,
  sortDir,
  onSort,
}: {
  products: Product[];
  sortKey: TableSortKey;
  sortDir: TableSortDir;
  onSort: (key: TableSortKey) => void;
}) {
  const theme = useHostTheme();
  const shellStyle = {
    overflowX: "auto" as const,
    border: `1px solid ${theme.stroke.primary}`,
    borderRadius: 6,
  };
  const headerStyle = {
    padding: "8px 12px",
    borderBottom: `1px solid ${theme.stroke.primary}`,
    background: theme.fill.secondary,
  };
  const cell = (flex: number, align: "left" | "right" = "left") => ({
    flex,
    minWidth: 0,
    textAlign: align,
  });

  return (
    <div style={shellStyle}>
      <Stack gap={0}>
        <Row gap={8} style={headerStyle}>
          {TABLE_COLUMNS.map((col) => {
            const active = sortKey === col.key;
            const indicator = active ? (sortDir === "asc" ? " ↑" : " ↓") : "";
            return (
              <div
                key={col.key}
                style={cell(columnFlex(col.key), col.numeric ? "right" : "left")}
              >
                <Button
                  variant="ghost"
                  onClick={() => onSort(col.key)}
                  style={{ fontWeight: active ? 600 : 400 }}
                >
                  {col.label}
                  {indicator}
                </Button>
              </div>
            );
          })}
        </Row>
        {products.map((product, index) => {
          const policyNames = product.pol ?? [];
          const rowStyle = {
            padding: "8px 12px",
            borderBottom: `1px solid ${theme.stroke.secondary}`,
            background: index % 2 === 1 ? theme.fill.tertiary : undefined,
          };
          return (
            <div key={product.n}>
              <Row gap={8} style={rowStyle} align="start">
                <Text size="small" style={cell(columnFlex("product"))}>{product.n}</Text>
                <Text size="small" style={cell(columnFlex("category"))}>
                  {DATA.cat[product.c] ?? product.c}
                </Text>
                <Text size="small" style={cell(columnFlex("auth"))}>{product.a}</Text>
                <Text size="small" style={{ ...cell(columnFlex("backends"), "right") }}>
                  {String(product.e.length)}
                </Text>
                <Text size="small" style={{ ...cell(columnFlex("apps"), "right") }}>
                  {String(product.p?.length ?? 0)}
                </Text>
                <Text size="small" style={{ ...cell(columnFlex("policies"), "right") }}>
                  {String(policyNames.length)}
                </Text>
                <Text
                  size="small"
                  tone={policyNames.length === 0 ? "tertiary" : "secondary"}
                  style={{ ...cell(columnFlex("policyNames")), lineHeight: 1.4 }}
                >
                  {formatPolicyChain(policyNames)}
                </Text>
              </Row>
            </div>
          );
        })}
      </Stack>
    </div>
  );
}

function compareProducts(a: Product, b: Product, key: TableSortKey): number {
  switch (key) {
    case "product":
      return a.n.localeCompare(b.n);
    case "category":
      return (DATA.cat[a.c] ?? a.c).localeCompare(DATA.cat[b.c] ?? b.c);
    case "auth":
      return a.a.localeCompare(b.a);
    case "backends":
      return a.e.length - b.e.length;
    case "apps":
      return (a.p?.length ?? 0) - (b.p?.length ?? 0);
    case "policies":
      return (a.pol?.length ?? 0) - (b.pol?.length ?? 0);
    case "policyNames":
      return formatPolicyChain(a.pol).localeCompare(formatPolicyChain(b.pol));
  }
}

function ProductGraph({
  product,
  backends,
  showApps,
}: {
  product: Product;
  backends: string[];
  showApps: boolean;
}) {
  const theme = useHostTheme();
  const backendIds = [...new Set(product.e.map(([idx]) => idx))];
  const apps = product.p ?? [];

  const nodes = [{ id: "product" }];
  const edges: { from: string; to: string }[] = [];

  if (showApps) {
    for (let i = 0; i < apps.length; i++) {
      nodes.push({ id: `a${i}` });
      edges.push({ from: "product", to: `a${i}` });
    }
  }
  for (const idx of backendIds) {
    nodes.push({ id: `b${idx}` });
    edges.push({ from: "product", to: `b${idx}` });
  }

  const layout = computeDAGLayout({
    nodes,
    edges,
    direction: "horizontal",
    nodeWidth: 168,
    nodeHeight: 36,
    rankGap: showApps && apps.length > 0 ? 120 : 96,
    nodeGap: 24,
    padding: 20,
  });

  const pathByBackend = new Map<number, string[]>();
  for (const [idx, path] of product.e) {
    const list = pathByBackend.get(idx) ?? [];
    list.push(path);
    pathByBackend.set(idx, list);
  }

  return (
    <Stack gap={12}>
      <div style={{ overflowX: "auto", width: "100%" }}>
        <svg
          width={Math.max(layout.width, 640)}
          height={Math.max(layout.height, 120)}
          style={{ display: "block" }}
        >
          {layout.edges.map((edge, i) => (
            <line
              key={i}
              x1={edge.sourceX}
              y1={edge.sourceY}
              x2={edge.targetX}
              y2={edge.targetY}
              stroke={theme.stroke.secondary}
              strokeWidth={1.5}
              markerEnd="url(#arrow)"
            />
          ))}
          <defs>
            <marker id="arrow" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto">
              <path d="M0,0 L0,6 L6,3 z" fill={theme.stroke.secondary} />
            </marker>
          </defs>
          {layout.nodes.map((node) => {
            const isProduct = node.id === "product";
            const isApp = node.id.startsWith("a");
            let label = node.id;
            let fill = theme.fill.secondary;
            let color = theme.text.primary;

            if (isProduct) {
              label = product.n;
              fill = theme.accent.control;
              color = theme.text.onAccent;
            } else if (isApp) {
              const appIdx = Number(node.id.slice(1));
              const app = apps[appIdx];
              label = app?.[0] ?? node.id;
              fill = theme.fill.tertiary;
            } else {
              label = backends[Number(node.id.slice(1))] ?? node.id;
            }

            return (
              <g key={node.id}>
                <rect
                  x={node.x}
                  y={node.y}
                  width={168}
                  height={36}
                  rx={4}
                  fill={fill}
                  stroke={theme.stroke.primary}
                />
                <text
                  x={node.x + 84}
                  y={node.y + 22}
                  textAnchor="middle"
                  fill={color}
                  fontSize={11}
                  fontFamily="var(--vscode-font-family, system-ui)"
                >
                  {label.length > 22 ? `${label.slice(0, 20)}…` : label}
                </text>
              </g>
            );
          })}
        </svg>
      </div>

      {showApps && apps.length > 0 ? (
        <Stack gap={8}>
          <Text weight="medium">Subscribed applications ({apps.length})</Text>
          {apps.map((app, i) => (
            <div key={`${app[0]}-${i}`}>
              <Text size="small" tone="secondary">
                {app[0]} · plan {app[1] || "—"} · {app[2] || "unknown"}
              </Text>
            </div>
          ))}
        </Stack>
      ) : null}

      {backendIds.length > 0 ? (
        <Stack gap={8}>
          <Text weight="medium">Routing paths</Text>
          {backendIds.map((idx) => (
            <div key={idx}>
              <Text size="small" tone="secondary">
                {backends[idx]}: {(pathByBackend.get(idx) ?? []).join(", ")}
              </Text>
            </div>
          ))}
        </Stack>
      ) : null}
    </Stack>
  );
}

export default function TopologyCanvas() {
  const [query, setQuery] = useCanvasState("query", "");
  const [selected, setSelected] = useCanvasState("selected", DATA.products[0]?.n ?? "");
  const [showApps, setShowApps] = useCanvasState("showApps", false);
  const [tablePage, setTablePage] = useCanvasState("tablePage", 0);
  const [pageSize, setPageSize] = useCanvasState("pageSize", 20);
  const [tableSortKey, setTableSortKey] = useCanvasState<TableSortKey>("tableSortKey", "backends");
  const [tableSortDir, setTableSortDir] = useCanvasState<TableSortDir>("tableSortDir", "desc");

  const handleSort = (key: TableSortKey) => {
    if (tableSortKey === key) {
      setTableSortDir(tableSortDir === "asc" ? "desc" : "asc");
    } else {
      setTableSortKey(key);
      const col = TABLE_COLUMNS.find((c) => c.key === key);
      setTableSortDir(col?.numeric ? "desc" : "asc");
    }
    setTablePage(0);
  };

  const filtered = DATA.products.filter((p) =>
    p.n.toLowerCase().includes(query.toLowerCase()),
  );
  const selectedProduct =
    DATA.products.find((p) => p.n === selected) ??
    filtered[0] ??
    DATA.products[0];

  const sortMultiplier = tableSortDir === "asc" ? 1 : -1;
  const sortedProducts = [...DATA.products].sort(
    (a, b) => compareProducts(a, b, tableSortKey) * sortMultiplier,
  );
  const size = Math.max(1, Number(pageSize) || 20);
  const totalPages = Math.max(1, Math.ceil(sortedProducts.length / size));
  const currentPage = Math.min(Math.max(0, tablePage), totalPages - 1);
  const pageStart = currentPage * size;
  const pageProducts = sortedProducts.slice(pageStart, pageStart + size);

  const pieData = Object.entries(DATA.catCounts).map(([key, value]) => ({
    label: DATA.cat[key] ?? key,
    value,
  }));

  const sharedCategories = DATA.shared.map((s) =>
    s.b.length > 18 ? `${s.b.slice(0, 16)}…` : s.b,
  );
  const sharedValues = DATA.shared.map((s) => s.n);

  return (
    <Stack gap={24} style={{ padding: 24, maxWidth: 1400 }}>
      <Stack gap={4}>
        <H1>3scale tenant — component topology</H1>
        <Text tone="secondary">
          Exported {DATA.m.exported_at} ·{" "}
          {DATA.m.admin_url}
        </Text>
      </Stack>

      <Row gap={16} wrap>
        <Stat label="API products" value={String(DATA.m.product_count)} tone="info" />
        <Stat label="Backends" value={String(DATA.m.backend_count)} tone="success" />
        <Stat label="Applications" value={String(DATA.m.application_count)} />
        <Stat
          label="Product→backend links"
          value={String(DATA.products.reduce((n, p) => n + p.e.length, 0))}
          tone="warning"
        />
      </Row>

      <Grid columns={2} gap={16}>
        <Card>
          <CardHeader>Products by domain</CardHeader>
          <CardBody>
            <PieChart data={pieData} size={220} />
            <Text tone="tertiary" size="small">
              {DATA.m.product_count} API products grouped by naming domain
            </Text>
          </CardBody>
        </Card>
        <Card>
          <CardHeader>Most shared backends</CardHeader>
          <CardBody>
            <BarChart
              categories={sharedCategories}
              series={[{ name: "Products referencing", data: sharedValues, tone: "info" }]}
              horizontal
              height={Math.max(220, sharedCategories.length * 28)}
            />
            <Text tone="tertiary" size="small">
              Backends used by more than one API product
            </Text>
          </CardBody>
        </Card>
      </Grid>

      <Card>
        <CardHeader>Product ↔ backend relationships</CardHeader>
        <CardBody>
          <Stack gap={16}>
            <Row gap={12} wrap align="end">
              <Stack gap={4} style={{ flex: 1, minWidth: 220 }}>
                <Text tone="secondary" size="small">Filter products</Text>
                <TextInput
                  value={query}
                  onChange={setQuery}
                  placeholder="Search by system name…"
                />
              </Stack>
              <Stack gap={4} style={{ minWidth: 280 }}>
                <Text tone="secondary" size="small">Selected product</Text>
                <Select
                  value={selectedProduct.n}
                  onChange={setSelected}
                  options={filtered.map((p) => ({ value: p.n, label: p.n }))}
                />
              </Stack>
            </Row>
            <Checkbox
              checked={showApps}
              onChange={setShowApps}
              label="Show subscribed applications"
            />
            <Row gap={8} wrap>
              <Text tone="secondary">
                Category: {DATA.cat[selectedProduct.c]} · Auth: {selectedProduct.a} ·{" "}
                {selectedProduct.e.length} backend(s)
                {showApps ? ` · ${selectedProduct.p?.length ?? 0} application(s)` : ""}
              </Text>
            </Row>
            <ProductGraph
              product={selectedProduct}
              backends={DATA.backends}
              showApps={showApps}
            />
          </Stack>
        </CardBody>
      </Card>

      <CollapsibleSection
        title="Top products by backend count"
        count={sortedProducts.length}
        defaultOpen
      >
        <Stack gap={12}>
          <Text tone="tertiary" size="small">
            All {sortedProducts.length} products · click a column header to sort
          </Text>
          <Row gap={12} wrap align="center">
            <Stack gap={4} style={{ minWidth: 140 }}>
              <Text tone="secondary" size="small">Rows per page</Text>
              <Select
                value={String(size)}
                onChange={(value) => {
                  setPageSize(Number(value));
                  setTablePage(0);
                }}
                options={PAGE_SIZE_OPTIONS}
              />
            </Stack>
            <Spacer />
            <Text tone="secondary" size="small">
              Page {currentPage + 1} of {totalPages}
            </Text>
            <Button
              variant="ghost"
              disabled={currentPage <= 0}
              onClick={() => setTablePage(currentPage - 1)}
            >
              Previous
            </Button>
            <Button
              variant="ghost"
              disabled={currentPage >= totalPages - 1}
              onClick={() => setTablePage(currentPage + 1)}
            >
              Next
            </Button>
          </Row>
          <ProductDataTable
            products={pageProducts}
            sortKey={tableSortKey}
            sortDir={tableSortDir}
            onSort={handleSort}
          />
        </Stack>
      </CollapsibleSection>

      <CollapsibleSection title="Shared backend detail" count={DATA.shared.length}>
        <Stack gap={12}>
          {DATA.shared.map((s) => (
            <div key={s.b}>
              <Card variant="borderless">
                <CardBody>
                  <Text weight="medium">{s.b}</Text>
                  <Text tone="secondary" size="small">
                    Referenced by {s.n} products: {s.p.join(", ")}
                    {s.n > s.p.length ? " …" : ""}
                  </Text>
                </CardBody>
              </Card>
            </div>
          ))}
        </Stack>
      </CollapsibleSection>
    </Stack>
  );
}
