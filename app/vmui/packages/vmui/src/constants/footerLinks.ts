import { CodeIcon, IssueIcon, WikiIcon } from "../components/Main/Icons";

const issueLink = {
  href: "https://github.com/VictoriaMetrics/VictoriaMetrics/issues/new/choose",
  Icon: IssueIcon,
  title: "Create an issue",
};

export const footerLinksByDefault = [
  {
    href: "https://docs.victoriametrics.com/victoriametrics/metricsql/",
    Icon: CodeIcon,
    title: "MetricsQL",
  },
  {
    href: "https://docs.victoriametrics.com/victoriametrics/single-server-victoriametrics/#vmui",
    Icon: WikiIcon,
    title: "Documentation",
  },
  issueLink
];

export const footerLinksToTraces = [
  {
    href: "https://docs.victoriametrics.com/victorialogs/logsql/",
    Icon: CodeIcon,
    title: "LogsQL",
  },
  {
    href: "https://docs.victoriametrics.com/victoriatraces/",
    Icon: WikiIcon,
    title: "Documentation",
  },
  issueLink
];
