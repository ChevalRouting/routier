import type { TypesLearnedRoute as LearnedRoute } from '@/api'
import { NodePositions } from '@/components/routing-overview/shared'
import { BGPPeerSessions } from '@/components/routing/overview/BGPPeerSessions'
import { groupBGPPeers } from '@/components/routing/overview/bgpPeers'
import { buildGraph, protoForNode } from '@/components/routing/overview/graph'
import { KernelRouteTable } from '@/components/routing/overview/KernelRouteTable'
import {
  edgeTypes,
  FitViewOnChange,
  nodeTypes,
} from '@/components/routing/overview/nodes'
import {
  RoutingConfig,
  Tunnel,
  WgIface,
} from '@/components/routing/overview/types'
import { api } from '@/lib/client'
import { BORDER, PRIMARY, topoNodeColor } from '@/lib/palette'
import { useFetch } from '@/lib/useFetch'
import {
  Background,
  Controls,
  MiniMap,
  type Node,
  type OnNodeDrag,
  type OnNodesChange,
  ReactFlow,
  applyNodeChanges,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { Badge, Card, CardContent, CardHeader, CardTitle, EmptyState, useTheme, Spinner, Table, TableBody, TableCell, TableHead, TableHeader, TableRow, Button } from 'cheval-ui'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'

interface TunnelInterface extends Tunnel { type?: string }

type RoutingOverviewShape4 = { hostname?: string }

type RoutingOverviewShape3 = { hostname?: string }

type RoutingOverviewShape2 = { table: number }

type RoutingOverviewShape = { table: number }

export default function RoutingOverview() {
	const navigate = useNavigate()
	const { theme } = useTheme()
	const { data: routing, isLoading } = useFetch<RoutingConfig>(
		() =>
			api.apiConfigSectionGet({ section: 'routing' }) as Promise<RoutingConfig>,
	)
	const { data: fullConfig } = useFetch<RoutingOverviewShape4>(
		() => api.apiConfigGet() as unknown as Promise<RoutingOverviewShape3>,
	)
	const { data: interfacesData } = useFetch<Record<string, TunnelInterface>>(
		() =>
			api.apiConfigSectionGet({ section: 'interfaces' }) as Promise<
				Record<string, TunnelInterface>
			>,
	)
	const { data: wireguardData } = useFetch<Record<string, WgIface>>(
		() =>
			api.apiConfigSectionGet({ section: 'vpn.wireguard' }) as Promise<
				Record<string, WgIface>
			>,
	)
	const { data: savedLayout } = useFetch<NodePositions>(
		() => api.apiUiLayoutGet() as Promise<NodePositions>,
	)
	const { data: statsData } = useFetch(() => api.apiStatsGet())
	const { data: learnedRoutes } = useFetch(() => api.apiRoutingLearnedGet())
	const { data: kernelRoutesData } = useFetch(() =>
		api.apiRoutingRoutesGet({ default_only: true }),
	)
	const { data: haStatus } = useFetch(() => api.apiHaStatusGet())
	const { data: vrfDeclarationsData } = useFetch<
		Record<string, RoutingOverviewShape2>
	>(
		() =>
			api.apiConfigSectionGet({ section: 'vrfs' }) as Promise<
				Record<string, RoutingOverviewShape>
			>,
	)

	const [selectedProto, setSelectedProto] = useState<string | null>('all')
	const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)
	const [learnedOpen, setLearnedOpen] = useState<Record<string, boolean>>({
		ospf: true,
		bgp: true,
	})
	const saveTimer = useRef<ReturnType<typeof setTimeout>>()
	const flowNodesRef = useRef<Node[]>([])
	const prevNodeIdsRef = useRef<string>('')

	const kernelRoutes = useMemo(() => kernelRoutesData?.routes ?? [], [kernelRoutesData])

	const bgpPeers = useMemo(
		() => groupBGPPeers(routing?.bgp?.neighbors ?? [], wireguardData ?? {}, statsData?.bgp?.peers ?? []),
		[routing, wireguardData, statsData],
	)

	const { nodes: baseNodes, edges } = useMemo(
		() =>
			buildGraph({
				routing: routing ?? {},
				tunnels: Object.fromEntries(Object.entries(interfacesData ?? {}).filter(([, iface]) => iface.type === 'tunnel')),
				wireguard: wireguardData ?? {},
				hostname: fullConfig?.hostname ?? 'router',
				bgpPeers,
				ospfNeighbors: statsData?.ospf?.neighbors ?? [],
				vrrpInstances: haStatus?.vrrp ?? [],
				kernelRoutes,
				vrfDeclarations: vrfDeclarationsData ?? {},
				selectedNodeId,
			}),
		[
			routing,
			interfacesData,
			wireguardData,
			fullConfig,
			bgpPeers,
			statsData,
			haStatus,
			kernelRoutes,
			vrfDeclarationsData,
			selectedNodeId,
		],
	)

	const [flowNodes, setFlowNodes] = useState<Node[]>([])
	const [fitVersion, setFitVersion] = useState(0)
	useEffect(() => {
		const newIds = baseNodes
			.map((n) => n.id)
			.sort()
			.join(',')
		const idsChanged = newIds !== prevNodeIdsRef.current
		prevNodeIdsRef.current = newIds

		const saved = savedLayout ?? {}
		const live: NodePositions = {}
		for (const n of flowNodesRef.current) live[n.id] = n.position
		const merged = { ...saved, ...live }

		const next =
			Object.keys(merged).length > 0
				? baseNodes.map((n) =>
						merged[n.id] ? { ...n, position: merged[n.id] } : n,
					)
				: baseNodes
		setFlowNodes(next)
		flowNodesRef.current = next

		if (idsChanged && baseNodes.length > 0) setFitVersion((v) => v + 1)
	}, [baseNodes, savedLayout])

	const onNodesChange: OnNodesChange = useCallback(
		(changes) =>
			setFlowNodes((nds) => {
				const next = applyNodeChanges(changes, nds)
				flowNodesRef.current = next
				return next
			}),
		[],
	)

	const onNodeDragStop: OnNodeDrag = useCallback(() => {
		const positions: NodePositions = {}
		for (const n of flowNodesRef.current) positions[n.id] = n.position
		clearTimeout(saveTimer.current)
		saveTimer.current = setTimeout(
			() => api.apiUiLayoutPut({ body: positions }),
			600,
		)
	}, [])

	const showAll = () => {
		setSelectedProto('all')
		setSelectedNodeId(null)
	}

	const resetLayout = useCallback(() => {
		flowNodesRef.current = []
		api.apiUiLayoutPut({ body: {} })
		setFitVersion((v) => v + 1)
	}, [])

	const onNodeClick = useCallback((_: React.MouseEvent, node: Node) => {
		const proto = protoForNode(node.id)
		if (!proto) return
		setSelectedProto((prev) => {
			if (prev === proto) {
				setSelectedNodeId(null)
				return 'all'
			}
			setSelectedNodeId(node.id)
			return proto
		})
	}, [])

	const onNodeDoubleClick = useCallback(
		(_: React.MouseEvent, node: Node) => {
			const dest: Record<string, string> = {
				bgp: '/routing',
				ospf: '/routing',
				static: '/routing',
				anycast: '/routing',
			}
			const proto = protoForNode(node.id)
			if (proto && dest[proto]) navigate(dest[proto])
		},
		[navigate],
	)

	const tunnelCount = Object.values(interfacesData ?? {}).filter((iface) => iface.type === 'tunnel').length
	const wgCount = Object.keys(wireguardData ?? {}).length

	const vrfCount = Object.keys(routing?.vrfs ?? {}).length

	const hasContent =
		(routing?.static?.length ?? 0) > 0 ||
		routing?.bgp != null ||
		routing?.ospf != null ||
		routing?.ospf6 != null ||
		(routing?.anycast?.services?.length ?? 0) > 0 ||
		tunnelCount > 0 ||
		wgCount > 0 ||
		vrfCount > 0 ||
		kernelRoutes.length > 0

	if (isLoading) {
		return (
			<div className="flex items-center justify-center h-64">
				<Spinner />
			</div>
		)
	}

	const selectedNodeTitle = selectedNodeId
		? ((flowNodes.find((n) => n.id === selectedNodeId)?.data?.nodeTitle as
				| string
				| undefined) ??
			selectedProto ??
			'')
		: (selectedProto ?? '')

	const selectedPeer = bgpPeers.find((p) => p.id === selectedNodeId)

	const frrProtoKey =
		selectedProto === 'ospf' || selectedProto === 'bgp' ? selectedProto : null
	const frrSelectedRoutes = frrProtoKey
		? (learnedRoutes?.[frrProtoKey] ?? [])
		: []

	const routeTableTitle =
		selectedProto === 'all'
			? 'All Routes'
			: selectedProto
				? `${selectedNodeTitle}, Routes${frrSelectedRoutes.length > 0 ? ` · FRR RIB (${frrSelectedRoutes.length})` : ''}`
				: ''

	return (
		<div className="space-y-6">
			<div className="flex flex-wrap gap-2">
				{routing?.bgp && (
					<Badge variant="secondary">
						BGP AS{routing.bgp.asn} · {routing.bgp.neighbors?.length ?? 0}{' '}
						neighbor{routing.bgp.neighbors?.length !== 1 ? 's' : ''}
					</Badge>
				)}
				{routing?.ospf && (
					<Badge variant="secondary">
						OSPF · {routing.ospf.areas?.length ?? 0} area
						{routing.ospf.areas?.length !== 1 ? 's' : ''}
						{(statsData?.ospf?.neighbors?.length ?? 0) > 0
							? ` · ${statsData!.ospf!.neighbors!.length} neighbor${statsData!.ospf!.neighbors!.length !== 1 ? 's' : ''}`
							: ''}
					</Badge>
				)}
				{routing?.ospf6 && (
					<Badge variant="secondary">
						OSPFv3 · {routing.ospf6.areas?.length ?? 0} area
						{routing.ospf6.areas?.length !== 1 ? 's' : ''}
					</Badge>
				)}
				{(routing?.static?.length ?? 0) > 0 && (
					<Badge variant="secondary">
						{routing!.static!.length} static route
						{routing!.static!.length !== 1 ? 's' : ''}
					</Badge>
				)}
				{kernelRoutes.length > 0 && (
					<Badge variant="secondary">
						{kernelRoutes.length} default route
						{kernelRoutes.length !== 1 ? 's' : ''}
					</Badge>
				)}
				{(routing?.anycast?.services?.length ?? 0) > 0 && (
					<Badge variant="secondary">
						Anycast · {routing!.anycast!.services!.length} service
						{routing!.anycast!.services!.length !== 1 ? 's' : ''}
					</Badge>
				)}
				{vrfCount > 0 && (
					<Badge variant="secondary">
						VRFs · {vrfCount}
					</Badge>
				)}
				{tunnelCount > 0 && (
					<Badge variant="secondary">
						{tunnelCount} tunnel{tunnelCount !== 1 ? 's' : ''}
					</Badge>
				)}
				{wgCount > 0 && (
					<Badge variant="secondary">
						WireGuard · {wgCount} interface{wgCount !== 1 ? 's' : ''}
					</Badge>
				)}
				{!hasContent && <Badge variant="outline">No routing configured</Badge>}
			</div>

			<Card>
				<CardHeader className="pb-4">
					<div className="flex items-center justify-between">
						<div>
							{selectedProto && selectedProto !== 'all' && (
								<Button variant="link" size="sm" className="h-auto p-0" onClick={showAll}>Show All</Button>
							)}
						</div>
						<Button variant="ghost" size="sm" onClick={resetLayout}>Reset Layout</Button>
					</div>
				</CardHeader>
				<CardContent className="p-0">
					<div className="h-[60dvh] min-h-80 md:h-[600px]">
						{hasContent ? (
							<ReactFlow
								nodes={flowNodes}
								edges={edges}
								nodeTypes={nodeTypes}
								edgeTypes={edgeTypes}
								onNodesChange={onNodesChange}
								onNodeDragStop={onNodeDragStop}
								onNodeClick={onNodeClick}
								onNodeDoubleClick={onNodeDoubleClick}
								fitView
								fitViewOptions={{ padding: 0.25 }}
								nodesDraggable
								nodesConnectable={false}
								colorMode={theme}
								proOptions={{ hideAttribution: true }}
							>
								<FitViewOnChange version={fitVersion} />
								<Background gap={16} color={BORDER} />
								<Controls showInteractive={false} />
								<MiniMap
									className="max-md:!hidden"
									nodeColor={(n) => n.id === 'router' ? PRIMARY : topoNodeColor(n.id)}
								/>
							</ReactFlow>
						) : (
							<div className="flex h-full items-center justify-center">
								<EmptyState
									className="border-0"
									title="No Routing Data"
								/>
							</div>
						)}
					</div>
				</CardContent>
			</Card>

			{selectedProto &&
				(() => {
					const frrRoutes: LearnedRoute[] = frrSelectedRoutes
					return (
						<Card>
							<CardHeader className="pb-2">
								<CardTitle className="text-sm">{routeTableTitle}</CardTitle>
							</CardHeader>
							<CardContent className="pt-0 space-y-4">
								{selectedPeer && <BGPPeerSessions peer={selectedPeer} />}
								<KernelRouteTable proto={selectedProto} />
								{frrRoutes.length > 0 && (
									<div className="space-y-1.5">
										<div className="text-xs text-muted-foreground font-medium">
											FRR RIB ({frrRoutes.length})
										</div>
										<div className="max-h-96 overflow-auto rounded-xl bg-card shadow-[var(--card-shadow)]">
											<Table className="w-full text-xs font-mono">
												<TableHeader className="sticky top-0 z-10 bg-card">
													<TableRow className="text-muted-foreground border-b border-border">
														{[
															'Prefix',
															'Next Hop',
															'Interface',
															'Metric',
															'Distance',
														].map((h) => (
															<TableHead
																key={h}
																className="px-3 py-2 text-left font-medium"
															>
																{h}
															</TableHead>
														))}
													</TableRow>
												</TableHeader>
												<TableBody>
													{frrRoutes.map((r, i) => (
														<TableRow
															key={i}
															className="border-b border-border/40 last:border-0 hover:bg-muted/30"
														>
															<TableCell className="px-3 py-1">{r.prefix}</TableCell>
															<TableCell className="px-3 py-1 text-muted-foreground">
																{r.nexthop ?? '-'}
															</TableCell>
															<TableCell className="px-3 py-1 text-muted-foreground">
																{r.interface ?? '-'}
															</TableCell>
															<TableCell className="px-3 py-1">{r.metric ?? '-'}</TableCell>
															<TableCell className="px-3 py-1">{r.distance ?? '-'}</TableCell>
														</TableRow>
													))}
												</TableBody>
											</Table>
										</div>
									</div>
								)}
							</CardContent>
						</Card>
					)
				})()}

			{hasContent && (
				<div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
					{routing?.bgp && (
						<Card>
							<CardHeader className="pb-2">
								<CardTitle className="text-sm">BGP</CardTitle>
							</CardHeader>
							<CardContent className="space-y-1.5 text-sm">
								<div className="flex justify-between">
									<span className="text-muted-foreground">Local ASN</span>
									<span className="font-mono">{routing.bgp.asn}</span>
								</div>
								<div className="flex justify-between">
									<span className="text-muted-foreground">Router ID</span>
									<span className="font-mono">
										{routing.bgp.router_id ?? '-'}
									</span>
								</div>
								<div className="flex justify-between">
									<span className="text-muted-foreground">Neighbors</span>
									<span>{routing.bgp.neighbors?.length ?? 0}</span>
								</div>
							</CardContent>
						</Card>
					)}
					{routing?.ospf && (
						<Card>
							<CardHeader className="pb-2">
								<CardTitle className="text-sm">OSPF</CardTitle>
							</CardHeader>
							<CardContent className="space-y-1.5 text-sm">
								<div className="flex justify-between">
									<span className="text-muted-foreground">Router ID</span>
									<span className="font-mono">
										{routing.ospf.router_id ?? '-'}
									</span>
								</div>
								<div className="flex justify-between">
									<span className="text-muted-foreground">Areas</span>
									<span>{routing.ospf.areas?.length ?? 0}</span>
								</div>
								<div className="flex justify-between">
									<span className="text-muted-foreground">Live Neighbors</span>
									<span>{statsData?.ospf?.neighbors?.length ?? 0}</span>
								</div>
							</CardContent>
						</Card>
					)}
					{routing?.ospf6 && (
						<Card>
							<CardHeader className="pb-2">
								<CardTitle className="text-sm">OSPFv3</CardTitle>
							</CardHeader>
							<CardContent className="space-y-1.5 text-sm">
								<div className="flex justify-between">
									<span className="text-muted-foreground">Router ID</span>
									<span className="font-mono">
										{routing.ospf6.router_id ?? '-'}
									</span>
								</div>
								<div className="flex justify-between">
									<span className="text-muted-foreground">Areas</span>
									<span>{routing.ospf6.areas?.length ?? 0}</span>
								</div>
							</CardContent>
						</Card>
					)}
					{vrfCount > 0 && (
						<Card>
							<CardHeader className="pb-2">
								<CardTitle className="text-sm">VRFs</CardTitle>
							</CardHeader>
							<CardContent className="space-y-1.5 text-sm">
								<div className="text-2xl font-bold">{vrfCount}</div>
								<p className="text-xs text-muted-foreground">
									{
										Object.values(routing?.vrfs ?? {}).filter(
											(v) => v.bgp || v.ospf || v.ospf6,
										).length
									}{' '}
									with routing protocols
								</p>
							</CardContent>
						</Card>
					)}
					{(routing?.static?.length ?? 0) > 0 && (
						<Card>
							<CardHeader className="pb-2">
								<CardTitle className="text-sm">Static Routes</CardTitle>
							</CardHeader>
							<CardContent>
								<div className="text-2xl font-bold">
									{routing!.static!.length}
								</div>
								<p className="text-xs text-muted-foreground">
									kernel-managed (proto routier)
								</p>
							</CardContent>
						</Card>
					)}
					{(routing?.anycast?.services?.length ?? 0) > 0 && (
						<Card>
							<CardHeader className="pb-2">
								<CardTitle className="text-sm">Anycast</CardTitle>
							</CardHeader>
							<CardContent>
								<div className="text-2xl font-bold">
									{routing!.anycast!.services!.length}
								</div>
								<p className="text-xs text-muted-foreground">
									{routing!.anycast!.services!.filter((s) => s.active).length}{' '}
									active
								</p>
							</CardContent>
						</Card>
					)}
					{tunnelCount > 0 && (
						<Card>
							<CardHeader className="pb-2">
								<CardTitle className="text-sm">Tunnels</CardTitle>
							</CardHeader>
							<CardContent>
								<div className="text-2xl font-bold">{tunnelCount}</div>
								<p className="text-xs text-muted-foreground">interfaces</p>
							</CardContent>
						</Card>
					)}
					{wgCount > 0 && (
						<Card>
							<CardHeader className="pb-2">
								<CardTitle className="text-sm">WireGuard</CardTitle>
							</CardHeader>
							<CardContent>
								<div className="text-2xl font-bold">{wgCount}</div>
								<p className="text-xs text-muted-foreground">
									{Object.values(wireguardData ?? {}).reduce(
										(s, w) => s + (w.peers?.length ?? 0),
										0,
									)}{' '}
									peers total
								</p>
							</CardContent>
						</Card>
					)}
				</div>
			)}

			{((learnedRoutes?.ospf?.length ?? 0) > 0 ||
				(learnedRoutes?.bgp?.length ?? 0) > 0) && (
				<div className="space-y-4">
					{(['ospf', 'bgp'] as const).map((proto) => {
						const list = learnedRoutes?.[proto] ?? []
						if (list.length === 0) return null
						const open = learnedOpen[proto] ?? false
						return (
							<Card key={proto}>
								<CardHeader
									className="pb-2 cursor-pointer select-none"
									onClick={() =>
										setLearnedOpen((v) => ({ ...v, [proto]: !v[proto] }))
									}
								>
									<div className="flex items-center gap-2">
										{open ? (
											<ChevronDown className="h-4 w-4" />
										) : (
											<ChevronRight className="h-4 w-4" />
										)}
										<CardTitle className="text-sm">
											{proto.toUpperCase()} Learned Routes
											<Badge variant="secondary" className="ml-2">
												{list.length}
											</Badge>
										</CardTitle>
									</div>
								</CardHeader>
								{open && (
									<CardContent className="pt-0">
										<Table className="w-full text-xs font-mono">
											<TableHeader>
												<TableRow className="text-muted-foreground border-b">
													<TableHead className="text-left py-1 pr-4">Prefix</TableHead>
													<TableHead className="text-left py-1 pr-4">Next Hop</TableHead>
													<TableHead className="text-left py-1 pr-4">Interface</TableHead>
													<TableHead className="text-left py-1 pr-4">Metric</TableHead>
													<TableHead className="text-left py-1">Distance</TableHead>
												</TableRow>
											</TableHeader>
											<TableBody>
												{list.map((r, i) => (
													<TableRow
														key={i}
														className="border-b border-border/40 hover:bg-muted/30"
													>
														<TableCell className="py-1 pr-4">{r.prefix}</TableCell>
														<TableCell className="py-1 pr-4 text-muted-foreground">
															{r.nexthop ?? '-'}
														</TableCell>
														<TableCell className="py-1 pr-4 text-muted-foreground">
															{r.interface ?? '-'}
														</TableCell>
														<TableCell className="py-1 pr-4">{r.metric ?? '-'}</TableCell>
														<TableCell className="py-1">{r.distance ?? '-'}</TableCell>
													</TableRow>
												))}
											</TableBody>
										</Table>
									</CardContent>
								)}
							</Card>
						)
					})}
				</div>
			)}
		</div>
	)
}
