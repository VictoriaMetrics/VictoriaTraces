// TraceSearchPage.tsx
import * as React from 'react';
import { Input, Button, Typography, Divider } from 'antd';
import type { InputRef } from 'antd';
import { connect } from 'react-redux';
import { bindActionCreators, Dispatch } from 'redux';
import memoizeOne from 'memoize-one';

import SearchForm, { TimeRange, DurationRange } from './SearchForm';
import { getUrlState, isSameQuery } from '../common/url';
import * as jaegerApiActions from '../../actions/jaeger-api';
import * as orderBy from '../../model/order-by';
import { sortTraces } from '../../model/search';
import LoadingIndicator from '../common/LoadingIndicator';
import ErrorMessage from '../common/ErrorMessage';
import { fetchedState } from '../../constants';

import withRouteProps from '../../utils/withRouteProps';

import type { ReduxState } from '../../types';
import type { SearchQuery } from '../../types/search';
import type { Trace } from '../../types/trace';
import type { IOtelTrace } from '../../types/otel';
import type { TUrlState } from '../common/url';

import './index.css'

// TODO: adjust import path to your project structure
import {
  TracePageImpl,
  mapStateToProps as traceMapState,
  mapDispatchToProps as traceMapDispatch,
} from './Trace';

const stripTraceKeys = (q: any) => {
  if (!q) return q;
  const { traceId, traceID, span, spanLinks, ...rest } = q;
  return rest;
};

const { Text } = Typography;

interface IServiceWithOperations {
  name: string;
  operations: string[];
}

interface IQueryOfResults extends Partial<SearchQuery> {
  service?: string;
  limit?: string | number;
}

interface IEmbeddedConfig {
  searchHideGraph?: boolean;
}

type TOwnProps = {
  history: any;
  location: any;
  timeRange: TimeRange;
  durationRange: DurationRange;
  resultsLimit: string;
};

type TStateProps = {
  embedded?: IEmbeddedConfig;

  loadingServices: boolean;
  loadingTraces: boolean;
  services: IServiceWithOperations[] | null;

  // search results (from reducer)
  traces: Trace[];
  traceResultsToDownload: unknown[];
  queryOfResults: IQueryOfResults | null;
  urlQueryParams: TUrlState | null;
  maxTraceDuration: number;
  errors: Array<{ message: string }> | null;

  sortedTracesXformer: (traces: Trace[], sortBy: string) => IOtelTrace[];

  isHomepage?: boolean;
};

type TDispatchProps = {
  fetchServiceOperations: (service: string) => void;
  fetchServices: () => void;
  searchTraces: (query: TUrlState) => void;

  // for traceId direct open
  fetchTrace: (id: string) => void;
};

type TProps = TOwnProps & TStateProps & TDispatchProps;

type TViewMode = 'search' | 'trace';

type TState = {
  sortBy: string;
  traceIdInput: string;
  viewMode: TViewMode;
  selectedTraceId: string | null;
};

function normalizeTraceId(id: string) {
  return (id || '').trim();
}

function guessTraceIdFromOtelTrace(t: IOtelTrace): string | null {
  return (
    (t as any).traceID ||
    (t as any).traceId ||
    (t as any).traceIdLower ||
    (t as any).id ||
    null
  );
}

/** ===== URL helpers for shareable state ===== */
function getTraceIdFromSearch(search: string): string | null {
  const sp = new URLSearchParams(search || '');
  const v = sp.get('traceId');
  return v ? v.trim() : null;
}

function setTraceIdToSearch(search: string, traceId: string | null): string {
  const sp = new URLSearchParams(search || '');
  if (traceId && traceId.trim()) sp.set('traceId', traceId.trim());
  else sp.delete('traceId');
  const next = sp.toString();
  return next ? `?${next}` : '';
}

/**
 * Lightweight inline results list.
 * Avoid reusing SearchResults if it navigates to /trace/:id internally.
 */
function SearchResultsInline(props: {
  loading: boolean;
  errors: Array<{ message: string }> | null;
  traces: IOtelTrace[];
  onPickTrace: (traceId: string) => void;
}) {
  const { loading, errors, traces, onPickTrace } = props;

  if (loading) return <LoadingIndicator className="u-mt-vast" centered />;

  if (errors && errors.length) {
    return (
      <div className="js-test-error-message" style={{ padding: 12 }}>
        <h2>There was an error loading traces: </h2>
        {errors.map(err => (
          <ErrorMessage key={err.message} error={err} />
        ))}
      </div>
    );
  }

  if (!traces.length) {
    return (
      <div style={{ padding: 12 }}>
        <Text type="secondary">No results.</Text>
      </div>
    );
  }

  return (
    <div style={{ padding: 12 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 10 }}>
        <Text strong>Results</Text>
        <Text type="secondary">({traces.length})</Text>
      </div>

      {traces.map((t, idx) => {
        const id = guessTraceIdFromOtelTrace(t);
        const duration = (t as any).duration;
        const startTime = (t as any).startTime;
        const serviceName = (t as any).processes?.[0]?.serviceName || (t as any).rootServiceName;

        return (
          <div
            key={`${id || 'unknown'}-${idx}`}
            style={{
              padding: '10px 10px',
              border: '1px solid #eee',
              borderRadius: 8,
              marginBottom: 8,
              display: 'flex',
              justifyContent: 'space-between',
              gap: 12,
            }}
          >
            <div style={{ minWidth: 0 }}>
              <div style={{ fontFamily: 'monospace', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                {id || '<no traceId>'}
              </div>
              <div style={{ marginTop: 4, display: 'flex', flexWrap: 'wrap', gap: 10 }}>
                {serviceName ? <Text type="secondary">service: {serviceName}</Text> : null}
                {typeof duration === 'number' ? <Text type="secondary">duration: {duration}</Text> : null}
                {startTime ? <Text type="secondary">start: {String(startTime)}</Text> : null}
              </div>
            </div>
            <div style={{ flexShrink: 0 }}>
              <Button type="primary" disabled={!id} onClick={() => id && onPickTrace(id)}>
                Open
              </Button>
            </div>
          </div>
        );
      })}
    </div>
  );
}

export class TraceSearchPageImpl extends React.PureComponent<TProps, TState> {
  state: TState = {
    sortBy: orderBy.MOST_RECENT,
    traceIdInput: '',
    viewMode: 'search',
    selectedTraceId: null,
  };

  private _traceIdInputRef = React.createRef<InputRef>();

  componentDidMount() {
    const { fetchServiceOperations, fetchServices, queryOfResults, searchTraces, urlQueryParams, history } = this.props;

    // keep original SearchTracePage behavior: if url has query and differs from current results, search by url
    // if (urlQueryParams && queryOfResults && !isSameQuery(urlQueryParams as any, queryOfResults as any)) {
    //   searchTraces(urlQueryParams);
    // }
    if (urlQueryParams && !isSameQuery(urlQueryParams as any, queryOfResults as any)) {
      searchTraces(urlQueryParams);
    }

    fetchServices();

    // lastSearch -> fetchServiceOperations
    const storedLastSearch = localStorage.getItem('lastSearch');
    const lastSearch = storedLastSearch ? (JSON.parse(storedLastSearch) as { service?: string }) : undefined;
    let { service } = lastSearch || {};
    if (urlQueryParams && urlQueryParams.service) {
      const urlService = urlQueryParams.service;
      if (typeof urlService === 'string') service = urlService;
      else if (Array.isArray(urlService)) service = urlService[0];
    }
    if (service && service !== '-') {
      fetchServiceOperations(service);
    }

    // ✅ NEW: URL shareable trace state
    const traceIdFromUrl = getTraceIdFromSearch(this.props.location.search);
    if (traceIdFromUrl) {
      this.setState({
        viewMode: 'trace',
        selectedTraceId: traceIdFromUrl,
        traceIdInput: traceIdFromUrl,
      });
      this.props.fetchTrace(traceIdFromUrl);
    }
  }

  componentDidUpdate(prevProps: TProps) {
    // Only react to URL search changes
    if (prevProps.location.search === this.props.location.search) return;

    const nextSearchStr = this.props.location.search;
    const prevSearchStr = prevProps.location.search;

    const nextTraceId = getTraceIdFromSearch(nextSearchStr);
    const prevTraceId = getTraceIdFromSearch(prevSearchStr);

    const nextUrlState = getUrlState(nextSearchStr);
    const prevUrlState = getUrlState(prevSearchStr);

    // Remove trace-related keys and only keep search-form-related params
    const nextSearchState = stripTraceKeys(nextUrlState);
    const prevSearchState = stripTraceKeys(prevUrlState);

    /**
     * Determine whether the URL contains any "effective" search parameters.
     * This avoids treating default-only params (e.g. lookback, limit)
     * as an intentional search.
     */
    const hasEffectiveSearchParam = (() => {
      if (!nextSearchState) return false;

      // Service is considered effective if it is present and not "-"
      const svc = (nextSearchState as any).service;
      const service =
        typeof svc === 'string' ? svc : Array.isArray(svc) ? svc[0] : undefined;
      if (service && service !== '-') return true;

      // Any other meaningful search fields also count as effective
      const keys = ['operation', 'tags', 'minDuration', 'maxDuration', 'start', 'end'];
      return keys.some(k => {
        const v = (nextSearchState as any)[k];
        // `true` may come from query-string when a key has no explicit value
        return v !== undefined && v !== null && v !== '' && v !== true;
      });
    })();

    /**
     * Priority rules:
     *
     * 1) If URL contains `traceId`, always switch to trace view and fetch the trace.
     * 2) If no `traceId` but URL contains effective search params, trigger a search.
     * 3) If neither exists, restore the pure initial (homepage) state.
     */

    // ===== 1) Trace view synchronization (highest priority) =====
    if (prevTraceId !== nextTraceId) {
      if (nextTraceId) {
        // Enter trace view
        if (
          this.state.viewMode !== 'trace' ||
          this.state.selectedTraceId !== nextTraceId ||
          this.state.traceIdInput !== nextTraceId
        ) {
          this.setState({
            viewMode: 'trace',
            selectedTraceId: nextTraceId,
            traceIdInput: nextTraceId,
          });
        }
        this.props.fetchTrace(nextTraceId);
      } else {
        // traceId was removed from URL
        // Do not decide the next view mode here;
        // let the search-param logic below handle it.
        if (this.state.viewMode === 'trace' || this.state.selectedTraceId) {
          this.setState({ viewMode: 'search', selectedTraceId: null });
        }
      }
    }

    // If traceId exists, do not allow search logic to override trace view
    if (nextTraceId) return;

    // ===== 2) Search query synchronization (only when no traceId) =====
    if (hasEffectiveSearchParam) {
      const currentComparable = stripTraceKeys(this.props.queryOfResults);

      // Trigger search only when URL search state differs from current results
      if (
        !currentComparable ||
        !isSameQuery(nextSearchState as any, currentComparable as any)
      ) {
        this.props.searchTraces(nextSearchState);
      }

      // Ensure we are in search view
      if (this.state.viewMode !== 'search') {
        this.setState({ viewMode: 'search', selectedTraceId: null });
      }
      return;
    }

    // ===== 3) Pure initial (homepage) state =====
    // No traceId and no effective search params:
    // restore the clean initial view without dispatching any search action
    if (this.state.viewMode !== 'search' || this.state.selectedTraceId) {
      this.setState({ viewMode: 'search', selectedTraceId: null });
    }
  }


  private pickTrace = (traceIdRaw: string) => {
    const traceId = normalizeTraceId(traceIdRaw);
    if (!traceId) return;

    // 1) write to URL for shareable state (keep other search params)
    const nextSearch = setTraceIdToSearch('', traceId);
    if (nextSearch !== this.props.location.search) {
      this.props.history.push(
        { pathname: this.props.location.pathname, search: nextSearch, hash: this.props.location.hash },
        this.props.location.state
      );
      setTimeout(() => console.log('[pickTrace] after push', window.location.href), 0);
    }

    // 2) local switch
    this.setState({ selectedTraceId: traceId, viewMode: 'trace', traceIdInput: traceId });

    // 3) fetch trace
    this.props.fetchTrace(traceId);
  };

  private onTraceIdEnter = () => {
    const traceId = normalizeTraceId(this.state.traceIdInput);
    if (!traceId) return;
    this.pickTrace(traceId);
  };

  private renderTraceIdBar() {
    return (
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: 8,
          alignItems: 'stretch',
          minWidth: 0,
        }}
      >
        {/* Label */}
        <Text strong style={{ whiteSpace: 'nowrap' }}>
          TraceId
        </Text>

        {/* Input */}
        <Input
          ref={this._traceIdInputRef}
          value={this.state.traceIdInput}
          placeholder="Paste traceId"
          onChange={e => this.setState({ traceIdInput: e.target.value })}
          onPressEnter={this.onTraceIdEnter}
          allowClear
          style={{
            width: '36ch', // enough for 32-hex traceId (+ a bit)
            fontFamily: 'monospace',
          }}
        />

        {/* Action */}
        <Button
          type="primary"
          style={{ alignSelf: 'flex-end' }}
          onClick={this.onTraceIdEnter}
        >
          Go
        </Button>
      </div>
    );
  }


  private renderTopBar() {
    const { loadingServices, services, timeRange, durationRange, resultsLimit } = this.props;

    return (
      <div style={{ display: 'flex', gap: 12, alignItems: 'stretch' }}>
        {/* Left: SearchForm fixed */}
        <div style={{ flex: 1, minWidth: 0 }}>
          {!loadingServices && services ? (
            <SearchForm
              services={services}
              timeRange={timeRange}
              durationRange={durationRange}
              resultsLimit={resultsLimit}
            />
          ) : <LoadingIndicator />}
        </div>

        <div
          style={{
            width: 1,
            backgroundColor: '#e5e5e5',
            margin: '0 8px',
          }}
        />

        {/* Right: traceId direct open */}
        <div style={{ flex: '0 0 auto', paddingTop: 8 }}>
          {this.renderTraceIdBar()}
        </div>
      </div>
    );
  }

  private renderBottom() {
    const { sortedTracesXformer, traces, loadingTraces, errors } = this.props;
    const { sortBy, viewMode, selectedTraceId } = this.state;

    if (viewMode === 'trace' && selectedTraceId) {
      const TraceInline = ConnectedTraceInline as any;
      return (
        <div style={{ paddingTop: 8 }}>
          <TraceInline history={this.props.history} location={this.props.location} params={{ id: selectedTraceId }} />
        </div>
      );
    }

    const traceResults = sortedTracesXformer(traces, sortBy);

    return (
      <SearchResultsInline loading={loadingTraces} errors={errors} traces={traceResults} onPickTrace={this.pickTrace} />
    );
  }

  render() {
    return (
      <div style={{ padding: 12 }}>
        {this.renderTopBar()}
        <Divider style={{ margin: '8px 0 0 0' }} />
        {this.renderBottom()}
      </div>
    );
  }
}

/** ===== selectors / xformers (copied from SearchTracePage, minimal change) ===== */

const stateTraceXformer = memoizeOne((stateTrace: ReduxState['trace']) => {
  const { traces: traceMap, search } = stateTrace;
  const { query, results, state, error: traceError } = search;

  const loadingTraces = state === fetchedState.LOADING;
  const traces = results.map(id => traceMap[id].data).filter((t): t is Trace => t !== undefined);
  const rawTraces = (stateTrace as any).rawTraces || [];
  const maxDuration = Math.max.apply(
    null,
    traces.map(tr => tr.duration)
  );
  return { traces, rawTraces, maxDuration, traceError, loadingTraces, query };
});

const sortedTracesXformer = memoizeOne((traces: Trace[], sortBy: string) => {
  const traceResults = traces.slice();
  sortTraces(traceResults, sortBy);
  return traceResults.map(t => t.asOtelTrace());
});

const stateServicesXformer = memoizeOne((stateServices: ReduxState['services']) => {
  const { loading: loadingServices, services: serviceList, operationsForService: opsBySvc, error: serviceError } =
    stateServices;

  const storedLastSearchRaw = localStorage.getItem('lastSearch');
  const selectedService = storedLastSearchRaw
    ? (JSON.parse(storedLastSearchRaw) as { service?: string }).service
    : undefined;
  if (
    selectedService &&
    serviceList &&
    serviceList.includes(selectedService) &&
    (!opsBySvc || !opsBySvc[selectedService] || opsBySvc[selectedService].length === 0)
  ) {
    return { loadingServices: true, services: serviceList, serviceError };
  }

  const services =
    serviceList &&
    serviceList.map(name => ({
      name,
      operations: (opsBySvc && opsBySvc[name]) || [],
    }));

  return { loadingServices, services, serviceError };
});

export function mapStateToProps(state: ReduxState): TStateProps {
  const { embedded, router, services: stServices, config } = state;

  const query = getUrlState(router.location.search);
  const {
    query: queryOfResults,
    traces,
    rawTraces,
    maxDuration,
    traceError,
    loadingTraces,
  } = stateTraceXformer(state.trace);

  const traceIdFromUrl = getTraceIdFromSearch(router.location.search);
  const searchPart = stripTraceKeys(query);
  const hasSearchParams = Object.keys(searchPart).length > 0;
  const isHomepage = !traceIdFromUrl && !hasSearchParams;

  const { loadingServices, services, serviceError } = stateServicesXformer(stServices);

  const errors: Array<{ message: string }> = [];
  if (traceError && typeof traceError === 'object' && 'message' in traceError) {
    errors.push({ message: (traceError as any).message });
  }
  if (serviceError) {
    if (typeof serviceError === 'string') errors.push({ message: serviceError });
    else if (typeof serviceError === 'object' && 'message' in serviceError) errors.push({ message: (serviceError as any).message });
  }

  if (isHomepage) {
    return {
      queryOfResults: null,
      embedded,
      loadingServices,
      loadingTraces: false,
      services: (services || null) as any,
      traces: [],
      traceResultsToDownload: [],
      errors: null,
      maxTraceDuration: 0,
      sortedTracesXformer,
      urlQueryParams: null,
      isHomepage: true,
    };
  }


  return {
    queryOfResults: (queryOfResults as any) || null,
    embedded,
    loadingServices,
    loadingTraces,
    services: (services || null) as any,
    traces,
    traceResultsToDownload: rawTraces,
    errors: errors.length ? errors : null,
    maxTraceDuration: maxDuration,
    sortedTracesXformer,
    urlQueryParams: Object.keys(query).length > 0 ? (query as any) : null,
    isHomepage,
  };
}

export function mapDispatchToProps(dispatch: Dispatch): TDispatchProps {
  const { fetchServiceOperations, fetchServices, searchTraces, fetchTrace } = bindActionCreators(
    jaegerApiActions as any,
    dispatch
  );
  return {
    fetchServiceOperations,
    fetchServices,
    searchTraces,
    fetchTrace,
  };
}

const connector = connect(mapStateToProps, mapDispatchToProps);
export default withRouteProps(connector(TraceSearchPageImpl));

/** ===== connect TracePageImpl for inline render ===== */
const ConnectedTraceInline = connect(traceMapState as any, traceMapDispatch as any)(TracePageImpl as any);
