import { FC, ReactNode, useCallback } from "preact/compat";
import Modal from "../Modal";
import "./style.scss";
import Tooltip from "../Tooltip";
import keyList from "./constants/keyList";
import { isMacOs } from "../../../utils/detect-device";
import useBoolean from "../../../hooks/useBoolean";
import useEventListener from "../../../hooks/useEventListener";

const title = "Shortcut keys";
const isMac = isMacOs();
const keyOpenHelp = isMac ? "F1 or Cmd + /" : "F1";

type Props = {
  children?: ReactNode
  withHotkey?: boolean
}

const ShortcutKeys: FC<Props> = ({ children, withHotkey = true }) => {

  const {
    value: openList,
    setTrue: handleOpen,
    setFalse: handleClose,
  } = useBoolean(false);

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (!withHotkey) return;
    const openOnF1 = e.key === "F1";
    const openOnMac = isMac && e.key === "/" && e.metaKey;
    if (openOnF1 || openOnMac) handleOpen();
  }, [handleOpen, withHotkey]);

  useEventListener("keydown", handleKeyDown);

  return <>
    <Tooltip
      title={withHotkey ? `${title} (${keyOpenHelp})` : title}
      placement="bottom-center"
    >
      <div onClick={handleOpen}>
        {children}
      </div>
    </Tooltip>

    {openList && (
      <Modal
        title={"Shortcut keys"}
        onClose={handleClose}
      >
        <div className="vm-shortcuts">
          {keyList.map(section => (
            <div
              className="vm-shortcuts-section"
              key={section.title}
            >
              <h3 className="vm-shortcuts-section__title">
                {section.title}
              </h3>
              <div className="vm-shortcuts-section-list">
                {section.list.map((l) => (
                  <div
                    className="vm-shortcuts-section-list-item"
                    key={l.description}
                  >
                    <div className="vm-shortcuts-section-list-item__key">
                      {l.keys}
                    </div>
                    <p className="vm-shortcuts-section-list-item__description">
                      {l.description}
                    </p>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </Modal>
    )}
  </>;
};

export default ShortcutKeys;
