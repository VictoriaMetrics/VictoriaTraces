type EmbeddedStateV0 = {
  version: 'v0';
  searchHideGraph: boolean;
  timeline: {
    collapseTitle: boolean;
    hideMinimap: boolean;
    hideSummary: boolean;
  };
};

export type EmbeddedState = EmbeddedStateV0;
