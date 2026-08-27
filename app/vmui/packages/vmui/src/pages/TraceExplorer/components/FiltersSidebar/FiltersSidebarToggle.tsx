import { useFiltersSidebarVisible } from "../../hooks/useFiltersSidebarVisible";
import { SidebarCloseIcon, SidebarOpenIcon } from "../../../../components/Main/Icons";
import Button from "../../../../components/Main/Button";
import useDeviceDetect from "../../../../hooks/useDeviceDetect";

const FiltersSidebarToggle = () => {
  const { isMobile } = useDeviceDetect();
  const { isVisible, setVisible } = useFiltersSidebarVisible();

  return (
    <Button
      variant="outlined"
      color={isVisible ? "gray" : "primary"}
      startIcon={isVisible ? <SidebarCloseIcon/> : <SidebarOpenIcon/>}
      onClick={() => setVisible(!isVisible)}
    >
      {!isMobile && `${isVisible ? "Hide" : "Show"} filters`}
    </Button>
  );
};

export default FiltersSidebarToggle;
