import type { ReactNode } from "react";
import Link from "@docusaurus/Link";
import useBaseUrl from "@docusaurus/useBaseUrl";
import useDocusaurusContext from "@docusaurus/useDocusaurusContext";
import Layout from "@theme/Layout";
import CodeBlock from "@theme/CodeBlock";

import dashboard from "../../../docs/screenshots/routier-dashboard.png";
import styles from "./index.module.css";

type Feature = {
	title: string;
	body: string;
};

type Audience = {
	title: string;
	body: string;
};

type TryPath = {
	title: string;
	body: string;
	to: string;
	action: string;
	code?: (repoUrl: string) => string;
};

const audiences: Audience[] = [
	{
		title: "Home and homelab",
		body: "A guided setup on first sign-in covers the Internet uplink and local networks, then every section stays one page away.",
	},
	{
		title: "Small offices and branch sites",
		body: "WireGuard between sites, a DHCP and DNS server, a stateful firewall and two-router failover, managed from a browser or a phone.",
	},
	{
		title: "Network engineers",
		body: "BGP, OSPF and BFD through FRR, VRFs, policy routing, sFlow export, managed switches, and every change available as YAML, CLI and API.",
	},
];

const features: Feature[] = [
	{
		title: "One declarative config",
		body: "Interfaces, routing, firewall and services live in a single YAML document. Routier renders it and applies only what changed.",
	},
	{
		title: "Rollback on every change",
		body: "Each apply takes a snapshot and arms a watchdog. If you lose access and do not confirm, the previous state comes back in 60 seconds.",
	},
	{
		title: "Web UI, API and CLI",
		body: "A responsive web UI and an HTTP API sit on the same engine as the routier command, so every action has three ways in.",
	},
	{
		title: "Full networking stack",
		body: "VLANs, bridges, bonds, VRFs, static and dynamic routing, WireGuard, Kea DHCP, BIND DNS and an nftables firewall.",
	},
	{
		title: "High availability",
		body: "Two routers share virtual addresses with VRRP and replicate connection state with conntrackd, so failover keeps sessions alive.",
	},
	{
		title: "Built on plain Alpine",
		body: "Install from an ISO, a cloud image or a signed apk package. Updates arrive through the same package manager as the rest of the system.",
	},
];

const tryPaths: TryPath[] = [
	{
		title: "In a virtual machine",
		body: "Boot the ISO with two network adapters, open the web UI and follow the setup. Your own network stays untouched.",
		to: "/docs/quickstart",
		action: "Try it in a VM",
	},
	{
		title: "On a cloud provider",
		body: "Build the qcow2 cloud image with task cloud and pass a password and a config through cloud-init user-data.",
		to: "/docs/installation#cloud-images",
		action: "Build the cloud image",
	},
	{
		title: "On an Alpine Linux box",
		body: "Add the signed repository and install the package. Networking is left alone until you apply and confirm.",
		to: "/docs/installation#package-install",
		action: "Install the package",
		code: (repoUrl: string) => `setup-apkrepos -c -1
wget -qO /etc/apk/keys/routier.rsa.pub ${repoUrl}/routier.rsa.pub
echo ${repoUrl}/nightly >> /etc/apk/repositories
apk update && apk add routier`,
	},
];

const exampleConfig = `version: v5.0.0

hostname: router

interfaces:
  wan:
    select: eth[0]
    addresses: [dhcp]
  lan:
    select: eth[1]
    addresses: [192.168.1.1/24]

dhcp:
  enabled: true
  subnets4:
    - subnet: 192.168.1.0/24
      interface: lan
      pools:
        - 192.168.1.100-192.168.1.200
      gateway: 192.168.1.1
      dns: [192.168.1.1]

nftables:
  chains:
    input:
      policy: drop
      rules: |
        iifname lo accept
        ct state established,related accept
        ct state invalid drop
        meta l4proto { icmp, icmpv6 } accept
        iifname $lan_interfaces tcp dport { 22, 8080 } accept
        iifname $lan_interfaces udp dport 67 accept
    forward:
      policy: drop
      rules: |
        ct state established,related accept
        iifname $lan_interfaces oifname $wan_interfaces accept
    postrouting:
      rules: |
        oifname $wan_interfaces masquerade

dns:
  nameservers: [127.0.0.1]
  server:
    enabled: true
    listen: [127.0.0.1, iface(lan)]
    allow_from: [127.0.0.1/32, 192.168.1.0/24]
    allow_inbound: [lan]
    upstreams: [1.1.1.1, 9.9.9.9]
`;

function Hero(): ReactNode {
	const { siteConfig } = useDocusaurusContext();
	const logo = useBaseUrl("/img/logo.svg");
	return (
		<header className={styles.hero}>
			<img className={styles.heroLogo} src={logo} alt="" />
			<h1 className={styles.heroTitle}>{siteConfig.title}</h1>
			<p className={styles.heroLead}>
				A router and firewall for Alpine Linux, configured from one YAML file,
				a web UI or an API, with a rollback safety net on every change.
			</p>
			<p className={styles.heroTagline}>
				{"Déjà vu, I've just been in this state before."}
			</p>
			<div className={styles.buttons}>
				<Link className="button button--primary button--lg" to="/docs/quickstart">
					Try it in a VM
				</Link>
				<Link className="button button--secondary button--lg" to="/docs/">
					Read the docs
				</Link>
			</div>
			<p className={styles.heroNote}>Free and open source, MIT licensed.</p>
		</header>
	);
}

function Showcase(): ReactNode {
	return (
		<section className={styles.showcase}>
			<img
				className={styles.screenshot}
				src={dashboard}
				alt="The Routier dashboard with interface status, resource usage and traffic charts"
			/>
		</section>
	);
}

function Audiences(): ReactNode {
	return (
		<section className={styles.section}>
			<h2 className={styles.sectionTitle}>Who it is for</h2>
			<div className={styles.grid3}>
				{audiences.map((audience) => (
					<div className={styles.card} key={audience.title}>
						<div className={styles.cardTitle}>{audience.title}</div>
						<p className={styles.cardBody}>{audience.body}</p>
					</div>
				))}
			</div>
		</section>
	);
}

function Features(): ReactNode {
	return (
		<section className={styles.section}>
			<h2 className={styles.sectionTitle}>What you get</h2>
			<div className={styles.grid3}>
				{features.map((feature) => (
					<div className={styles.card} key={feature.title}>
						<div className={styles.cardTitle}>{feature.title}</div>
						<p className={styles.cardBody}>{feature.body}</p>
					</div>
				))}
			</div>
		</section>
	);
}

function Example(): ReactNode {
	return (
		<section className={styles.example}>
			<h2 className={styles.sectionTitle}>A whole router, in one file</h2>
			<p className={styles.sectionLead}>
				The web UI writes the same document you can edit by hand. Run{" "}
				<code>routier plan</code> to see what changes and{" "}
				<code>routier apply</code> to apply it.
			</p>
			<CodeBlock language="yaml" title="/etc/routier/config.yml">
				{exampleConfig}
			</CodeBlock>
			<p className={styles.exampleLink}>
				<Link to="/docs/configuration">See the configuration model →</Link>
			</p>
		</section>
	);
}

function useSiteUrls(): { repoUrl: string; githubUrl: string } {
	const { siteConfig } = useDocusaurusContext();
	return siteConfig.customFields as { repoUrl: string; githubUrl: string };
}

function TryIt(): ReactNode {
	const { repoUrl } = useSiteUrls();
	return (
		<section className={styles.section}>
			<h2 className={styles.sectionTitle}>Three ways to try it</h2>
			<div className={styles.grid3}>
				{tryPaths.map((path) => (
					<div className={`${styles.card} ${styles.tryCard}`} key={path.title}>
						<div className={styles.cardTitle}>{path.title}</div>
						<p className={styles.cardBody}>{path.body}</p>
						{path.code && (
							<CodeBlock language="bash" className={styles.tryCode}>
								{path.code(repoUrl)}
							</CodeBlock>
						)}
						<Link className={styles.tryLink} to={path.to}>
							{path.action} →
						</Link>
					</div>
				))}
			</div>
		</section>
	);
}

function HelpTest(): ReactNode {
	const { githubUrl } = useSiteUrls();
	return (
		<section className={styles.callout}>
			<h2 className={styles.sectionTitle}>Help shape Routier</h2>
			<p className={styles.sectionLead}>
				Routier is young and moves quickly. Reports from hardware, networks and
				workflows we do not have are the most useful contribution right now,
				whether it is a crash or a label that did not make sense.
			</p>
			<div className={styles.buttons}>
				<Link
					className="button button--primary button--lg"
					href={`${githubUrl}/issues/new/choose`}
				>
					Report an issue
				</Link>
			</div>
		</section>
	);
}

export default function Home(): ReactNode {
	const { siteConfig } = useDocusaurusContext();
	return (
		<Layout title={siteConfig.title} description={siteConfig.tagline}>
			<Hero />
			<main>
				<Showcase />
				<Audiences />
				<TryIt />
				<Features />
				<Example />
				<HelpTest />
			</main>
		</Layout>
	);
}
