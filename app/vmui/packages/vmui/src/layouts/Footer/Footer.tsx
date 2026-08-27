import { FC, memo, useEffect, useRef } from "preact/compat";
import { LogoShortIcon } from "../../components/Main/Icons";
import "./style.scss";
import { footerLinksToTraces } from "../../constants/footerLinks";
import useGetVersion from "../../hooks/useGetVersion";
import { useResizeObserver } from "../../hooks/useResizeObserver";
import { setCssVariable } from "../../utils/theme";

interface Props {
  links?: {
    href: string;
    Icon: FC;
    title: string;
  }[]
}

const copyrightYears = `2019-${new Date().getFullYear()}`;

const Footer: FC<Props> = memo(({ links = footerLinksToTraces }) => {
  const { version } = useGetVersion();

  const footerRef = useRef<HTMLElement>(null);
  const { height: footerHeight } = useResizeObserver({ ref: footerRef });

  useEffect(() => {
    setCssVariable("footer-height", `${footerHeight || 0}px`);
    return () => setCssVariable("footer-height", "0px");
  }, [footerHeight]);

  return <footer
    id="vm-footer"
    className="vm-footer"
    ref={footerRef}
  >
    <a
      className="vm-link vm-footer__website"
      target="_blank"
      href="https://victoriametrics.com/"
      rel="me noreferrer"
    >
      <LogoShortIcon/>
      victoriametrics.com
    </a>
    {links.map(({ href, Icon, title }) => (
      <a
        className="vm-link vm-footer__link"
        target="_blank"
        href={href}
        rel="help noreferrer"
        key={`${href}-${title}`}
      >
        <Icon/>
        {title}
      </a>
    ))}
    <div className="vm-footer__copyright">&copy; {copyrightYears} VictoriaMetrics.</div>
    {version && <span className="vm-footer__version">&nbsp;Version:
      <a
        href={`https://github.com/VictoriaMetrics/VictoriaTraces/releases/tag/${version}`}
        target="_blank"
        rel="noreferrer"
      >{version}</a>
    </span>}
  </footer>;
});

export default Footer;
