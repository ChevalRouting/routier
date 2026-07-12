import type { ReactNode } from "react";
import Link from "@docusaurus/Link";
import useDocusaurusContext from "@docusaurus/useDocusaurusContext";
import Layout from "@theme/Layout";
import CodeBlock from "@theme/CodeBlock";

import styles from "./index.module.css";

type Feature = {
	title: string;
	body: string;
};

const features: Feature[] = [
	{
		title: "Declarative config",
		body: "Describe networking, routing, firewall, and services in a single YAML document. Routier renders and applies the state you want.",
	},
	{
		title: "Safe apply with rollback",
		body: "Every apply snapshots the previous state and arms a rollback watchdog, so even a bad firewall rule cannot lock you out of a remote box.",
	},
	{
		title: "UI, API, and CLI",
		body: "A web UI and an HTTP API sit on top of the same engine, and everything they can do has a routier CLI equivalent.",
	},
	{
		title: "Full networking stack",
		body: "Interfaces, VLANs, bridges, VRFs, static and dynamic routing (BGP, OSPF, BFD), WireGuard tunnels, and an nftables firewall.",
	},
	{
		title: "Friends",
		body: "Mutually-authenticated peer routers that exchange liveness and derive configuration from each other's configs over an encrypted channel.",
	},
	{
		title: "Runs on plain Alpine",
		body: "Install the apk package or boot the prebuilt ISO to turn a fresh Alpine Linux box into a router and firewall appliance.",
	},
];

const exampleConfig = `version: v3.0.0

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
        icmp type echo-request accept
        icmpv6 type { echo-request, nd-neighbor-solicit, nd-neighbor-advert, nd-router-solicit } accept
        iifname $lan_interfaces ip saddr $lan_network tcp dport { 22, 8080 } accept comment "Allow Routier UI and SSH from lan"
        iifname $lan_interfaces udp dport 67 accept
    forward:
      policy: drop
      rules: |
        ct state established,related accept
        ct state invalid drop
        iifname $lan_interfaces oifname $wan_interfaces accept
    postrouting:
      rules: |
        oifname $wan_interfaces masquerade

sysctl:
  net.ipv4.ip_forward: "1"
  net.ipv6.conf.all.forwarding: "1"

dns:
  nameservers: [1.1.1.1, 9.9.9.9]

ssh:
  port: 22
  permit_root_login: "no"
  password_auth: "no"

users:
  admin:
    uid: 1000
    shell: /bin/ash
    groups: [wheel]
    ssh_keys:
      - "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExample alice@home"
`;

function Hero(): ReactNode {
	const { siteConfig } = useDocusaurusContext();
	return (
		<header className={styles.hero}>
			<img className={styles.heroLogo} src="/img/logo.svg" alt="Routier" />
			<h1 className={styles.heroTitle}>{siteConfig.title}</h1>
			<p className={styles.heroTagline}>
				{"Déjà vu, I've just been in this state before."}
			</p>
			<div className={styles.buttons}>
				<Link className="button button--primary button--lg" to="/docs/">
					Get started
				</Link>
				<Link
					className="button button--secondary button--lg"
					to="/docs/installation"
				>
					Install
				</Link>
			</div>
		</header>
	);
}

export default function Home(): ReactNode {
	const { siteConfig } = useDocusaurusContext();
	return (
		<Layout title={siteConfig.title} description={siteConfig.tagline}>
			<Hero />
			<main>
				<section className={styles.example}>
					<h2 className={styles.sectionTitle}>A whole router, in one file</h2>
					<p className={styles.sectionLead}>
						Interfaces, DHCP, Nftables, DNS, SSH, BGP, OSPF and more in a single
						YAML document. Run <code>routier apply</code> and it renders and
						applies the state, with a snapshot and rollback watchdog on every
						change.
					</p>
					<CodeBlock language="yaml" title="config.yml">
						{exampleConfig}
					</CodeBlock>
					<p className={styles.exampleLink}>
						<Link to="/docs/configuration">
							See the full configuration model →
						</Link>
					</p>
				</section>

				<section className={styles.features}>
					{features.map((feature) => (
						<div className={styles.card} key={feature.title}>
							<div className={styles.cardTitle}>{feature.title}</div>
							<p className={styles.cardBody}>{feature.body}</p>
						</div>
					))}
				</section>
			</main>
		</Layout>
	);
}
