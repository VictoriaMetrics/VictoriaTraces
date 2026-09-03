import { FC, useMemo } from "preact/compat";
import { AutocompleteOptions } from "../Autocomplete";
import useDeviceDetect from "../../../../hooks/useDeviceDetect";
import { sanitizeHtml } from "../../../../utils/sanitizeHtml";
import "./style.scss";

type Props = {
  option?: AutocompleteOptions
}

const AutocompleteDetailsPanel: FC<Props> = ({ option }) => {
  const { isMobile } = useDeviceDetect();
  const { description, type } = option || {};
  const sanitizedDescription = useMemo(() => sanitizeHtml(description || ""), [description]);

  if (!description || isMobile) return null;

  return (
    <div className="vm-autocomplete-details-panel">
      <div className="vm-autocomplete__title">{type}</div>
      <div
        className="vm-autocomplete-details-panel__description vm-markdown"
        // eslint-disable-next-line @eslint-react/dom-no-dangerously-set-innerhtml -- content is passed through sanitizeHtml before rendering
        dangerouslySetInnerHTML={{ __html: sanitizedDescription }}
      />
    </div>
  );
};

export default AutocompleteDetailsPanel;
