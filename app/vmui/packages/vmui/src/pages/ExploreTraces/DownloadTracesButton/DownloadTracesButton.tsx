import { FC, useMemo } from "preact/compat";
import { useCallback } from "react";
import dayjs from "dayjs";
import DownloadButton from "../../../components/DownloadButton/DownloadButton";
import { DATE_FILENAME_FORMAT } from "../../../constants/date";
import { downloadCSV, downloadJSON } from "../../../utils/file";
import { Traces } from "../../../api/types";

interface DownloadTracesButtonProps {
  /** Callback to get traces to download */
  getTraces: () => Traces[];
}

const DownloadTracesButton: FC<DownloadTracesButtonProps> = ({ getTraces }) => {
  const { fileExtensions, getDownloaderByExtension } = useMemo(() => {
    const downloadFileOptions: {
      extension: string;
      downloader: (data: Record<string,string>[], filename: string) => void;
    }[] = [
      { extension: "csv", downloader: downloadCSV },
      {
        extension: "json",
        downloader: (data: Record<string,string>[], filename: string) => {
          const json = JSON.stringify(data, null, 2);
          downloadJSON(json, filename);
        }
      }
    ];

    const getDownloaderByExtension = (extension: string) => {
      return downloadFileOptions.find(({ extension: optionExtension }) => optionExtension === extension)?.downloader;
    };
    const fileExtensions = downloadFileOptions.map(({ extension }) => extension);

    return { fileExtensions, getDownloaderByExtension };
  }, []);

  const onDownload = useCallback((fileExtension?: string) => {
    if (!fileExtension){
      return;
    }

    const traces = getTraces();
    const downloader = getDownloaderByExtension(fileExtension);
    if (downloader){
      const timestamp = dayjs().utc().format(DATE_FILENAME_FORMAT);
      downloader(traces, `vmui_traces_${timestamp}.${fileExtension}`);
    }
  }, [getTraces]);

  return <DownloadButton
    title={"Download traces"}
    onDownload={onDownload}
    downloadFormatOptions={fileExtensions}
  />;
};

export default DownloadTracesButton;
