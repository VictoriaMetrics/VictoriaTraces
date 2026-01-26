import React, { useState, useCallback, useEffect } from 'react';
import { Input, Button, Popover, Select, Row, Col, Form, Switch } from 'antd';
import _get from 'lodash/get';
import logfmtParser from 'logfmt/lib/logfmt_parser';
import { stringify as logfmtStringify } from 'logfmt/lib/stringify';
import dayjs from 'dayjs';
import queryString from 'query-string';
import { IoHelp } from 'react-icons/io5';
import { connect, ConnectedProps } from 'react-redux';
import { bindActionCreators, Dispatch } from 'redux';

import * as jaegerApiActions from '../../actions/jaeger-api';
import {
  DEFAULT_OPERATION,
  CHANGE_SERVICE_ACTION_TYPE,
} from '../../constants/search-form';
import SearchableSelect from '../common/SearchableSelect';
import './SearchForm.css';
import { ReduxState } from '../../types';
import { SearchQuery } from '../../types/search';

const FormItem = Form.Item;
const Option = Select.Option;

const ADJUST_TIME_ENABLED_KEY = 'jaeger-ui/search-adjust-time-enabled';

export function convTagsLogfmt(tags: string | null | undefined): string | null {
  if (!tags) {
    return null;
  }
  const data = logfmtParser.parse(tags);
  Object.keys(data).forEach(key => {
    const value = data[key];
    // make sure all values are strings
    // https://github.com/jaegertracing/jaeger/issues/550#issuecomment-352850811
    if (typeof value !== 'string') {
      data[key] = String(value);
    }
  });
  return JSON.stringify(data);
}

function lookbackToTimestamp(lookback: string, from: Date | number): number {
  const unit = lookback.substr(-1) as any; // dayjs ManipulateType
  return dayjs(from).subtract(parseInt(lookback, 10), unit).valueOf() * 1000;
}

export function traceIDsToQuery(traceIDs: string | null | undefined): string[] | null {
  if (!traceIDs) {
    return null;
  }
  return traceIDs.split(',');
}

export function applyAdjustTime(endTimestamp: number, adjustTime: string | null | undefined): number {
  if (!adjustTime) {
    return endTimestamp;
  }
  const adjustedEnd = lookbackToTimestamp(adjustTime, endTimestamp / 1000);
  return adjustedEnd;
}

interface ISearchFormFields {
  service: string;
  operation: string;
  tags?: string;
}

export interface TimeRange {
  start: number; // microseconds
  end: number; // microseconds
}

export interface DurationRange {
  minDuration: string | null;
  maxDuration: string | null;
}

type SearchTracesFunction = typeof jaegerApiActions.searchTraces;

export function submitForm(
  fields: ISearchFormFields,
  timeRange: TimeRange,
  durationRange: DurationRange,
  resultsLimit: string,
  searchTraces: SearchTracesFunction,
  adjustTime: string | null | undefined,
  adjustTimeEnabled: boolean
): void {
  const { service, operation, tags } = fields;
  // Note: traceID is ignored when the form is submitted
  localStorage.setItem('lastSearch', JSON.stringify({ service, operation }));

  let { end } = timeRange;
  const { start } = timeRange;

  // Apply time adjustment to exclude very recent traces that may be incomplete
  if (adjustTimeEnabled) {
    end = applyAdjustTime(end, adjustTime);
  }

  searchTraces({
    service,
    operation: operation !== DEFAULT_OPERATION ? operation : undefined,
    limit: resultsLimit,
    // The time range is always resolved externally (vmui's global time picker), so
    // this is only a label the API expects, not something used for filtering.
    lookback: 'custom',
    start: String(start),
    end: String(end),
    tags: convTagsLogfmt(tags) || undefined,
    minDuration: durationRange.minDuration,
    maxDuration: durationRange.maxDuration,
  } as SearchQuery);
}

interface IServiceWithOperations {
  name: string;
  operations?: string[];
}

interface ISearchFormImplProps {
  invalid?: boolean;
  submitting?: boolean;
  searchAdjustEndTime?: string;
  useOtelTerms?: boolean;
  services: IServiceWithOperations[];
  initialValues?: Partial<ISearchFormFields> & { traceIDs?: string | null };
  timeRange: TimeRange;
  durationRange: DurationRange;
  resultsLimit: string;
  searchTraces: SearchTracesFunction;
  changeServiceHandler: (service: string) => void;
  submitFormHandler: (
    fields: ISearchFormFields,
    timeRange: TimeRange,
    durationRange: DurationRange,
    resultsLimit: string,
    adjustEndTime: string | null | undefined,
    adjustTimeEnabled: boolean
  ) => void;
}

export const SearchFormImpl: React.FC<ISearchFormImplProps> = ({
  invalid = false,
  submitting = false,
  searchAdjustEndTime,
  useOtelTerms,
  services = [],
  initialValues,
  timeRange,
  durationRange,
  resultsLimit,
  changeServiceHandler,
  submitFormHandler,
}) => {
  const [formData, setFormData] = useState<Partial<ISearchFormFields>>(() => ({
    service: initialValues?.service,
    operation: initialValues?.operation,
    tags: initialValues?.tags,
  }));

  useEffect(() => {
    // 只在 “initialValues 真的变化” 时覆盖表单
    setFormData({
      service: initialValues?.service,
      operation: initialValues?.operation,
      tags: initialValues?.tags,
    });
  }, [
    initialValues?.service,
    initialValues?.operation,
    initialValues?.tags,
  ]);

  const [adjustTimeEnabled, setAdjustTimeEnabled] = useState<boolean>(() => {
    const storedAdjustTimeEnabled = localStorage.getItem(ADJUST_TIME_ENABLED_KEY);
    return storedAdjustTimeEnabled !== null ? JSON.parse(storedAdjustTimeEnabled) : Boolean(searchAdjustEndTime);
  });

  const handleChange = useCallback(
    (fieldData: Partial<ISearchFormFields>) => {
      setFormData(prev => {
        const nextFormData = { ...prev, ...fieldData };
        if (fieldData.service) {
          changeServiceHandler(fieldData.service);
          nextFormData.operation = DEFAULT_OPERATION;
        }
        return nextFormData;
      });
    },
    [changeServiceHandler]
  );

  const handleAdjustTimeToggle = useCallback((checked: boolean) => {
    setAdjustTimeEnabled(checked);
    localStorage.setItem(ADJUST_TIME_ENABLED_KEY, JSON.stringify(checked));
  }, []);

  const handleSubmit = useCallback(
    (e: React.FormEvent<HTMLFormElement>) => {
      e.preventDefault();
      submitFormHandler(
        formData as ISearchFormFields,
        timeRange,
        durationRange,
        resultsLimit,
        searchAdjustEndTime,
        adjustTimeEnabled
      );
    },
    [formData, timeRange, durationRange, resultsLimit, searchAdjustEndTime, adjustTimeEnabled, submitFormHandler]
  );

  const { service: selectedService } = formData;
  const selectedServicePayload = services.find(s => s.name === selectedService);
  const opsForSvc = (selectedServicePayload && selectedServicePayload.operations) || [];
  const noSelectedService = selectedService === '-' || !selectedService;

  return (
    <Form layout="vertical" onSubmitCapture={handleSubmit} className="SearchForm--compact">
      {/* Row 1: Service / Operation / Tags */}
      <Row gutter={16}>
        <Col span={8}>
          <FormItem
            label={
              <span>
                Service <span className="SearchForm--labelCount">({services.length})</span>
              </span>
            }
          >
            <SearchableSelect
              data-testid="service"
              value={formData.service}
              placeholder="Select A Service"
              disabled={submitting}
              onChange={(value: string) => handleChange({ service: value })}
            >
              {services.map(service => (
                <Option key={service.name} value={service.name}>
                  {service.name}
                </Option>
              ))}
            </SearchableSelect>
          </FormItem>
        </Col>

        <Col span={8}>
          <FormItem
            label={
              <span>
                {useOtelTerms ? 'Span Name' : 'Operation'}{' '}
                <span className="SearchForm--labelCount">({opsForSvc ? opsForSvc.length : 0})</span>
              </span>
            }
          >
            <SearchableSelect
              data-testid="operation"
              value={formData.operation}
              disabled={submitting || noSelectedService}
              placeholder={useOtelTerms ? 'Select A Span Name' : 'Select An Operation'}
              onChange={(value: string) => handleChange({ operation: value })}
            >
              {['all'].concat(opsForSvc).map(op => (
                <Option key={op} value={op}>
                  {op}
                </Option>
              ))}
            </SearchableSelect>
          </FormItem>
        </Col>

        <Col span={8}>
          <FormItem
            label={
              <div>
                {useOtelTerms ? 'Attributes' : 'Tags'}{' '}
                <Popover
                  placement="topLeft"
                  trigger="click"
                  title={
                    <h3 key="title" className="SearchForm--tagsHintTitle">
                      Values should be in the{' '}
                      <a href="https://brandur.org/logfmt" rel="noopener noreferrer" target="_blank">
                        logfmt
                      </a>{' '}
                      format.
                    </h3>
                  }
                  content={
                    <div>
                      <ul key="info" className="SearchForm--tagsHintInfo">
                        <li>Use space for AND conjunctions.</li>
                        <li>
                          Values containing whitespace or equal-sign &apos;=&apos; should be enclosed in quotes.
                        </li>
                        <li>
                          Elasticsearch/OpenSearch storage supports regex query, therefore{' '}
                          <a
                            href="https://lucene.apache.org/core/9_0_0/core/org/apache/lucene/util/automaton/RegExp.html"
                            rel="noopener noreferrer"
                            target="_blank"
                          >
                            reserved characters
                          </a>{' '}
                          need to be escaped for exact match queries.
                        </li>
                      </ul>
                      <p>Examples:</p>
                      <ul className="SearchForm--tagsHintInfo">
                        <li>
                          <code className="SearchForm--tagsHintEg">error=true</code>
                        </li>
                        <li>
                          <code className="SearchForm--tagsHintEg">
                            db.statement=&quot;select * from User&quot;
                          </code>
                        </li>
                        <li>
                          <code className="SearchForm--tagsHintEg">
                            http.url=&quot;http://0.0.0.0:8081/customer\\?customer=123&quot;
                          </code>
                          <div>
                            Note: when using Elasticsearch/OpenSearch the{' '}
                            <a
                              href="https://lucene.apache.org/core/9_0_0/core/org/apache/lucene/util/automaton/RegExp.html"
                              rel="noopener noreferrer"
                              target="_blank"
                            >
                              regex-reserved
                            </a>{' '}
                            character <code className="SearchForm--tagsHintEg">&quot;?&quot;</code> must be
                            escaped with <code className="SearchForm--tagsHintEg">&quot;\\&quot;</code>.
                          </div>
                        </li>
                      </ul>
                    </div>
                  }
                >
                  <IoHelp className="SearchForm--hintTrigger" />
                </Popover>
              </div>
            }
          >
            <Input
              name="tags"
              value={formData.tags}
              disabled={submitting}
              placeholder="http.status_code=200 error=true"
              onChange={e => handleChange({ tags: e.target.value })}
              allowClear
            />
          </FormItem>
        </Col>
      </Row>

      {searchAdjustEndTime && (
        <div className="SearchForm--adjustTime">
          <span className="SearchForm--adjustTimeLabel">Adjusted -{searchAdjustEndTime}</span>
          <Switch
            size="small"
            checked={adjustTimeEnabled}
            onChange={handleAdjustTimeToggle}
            disabled={submitting}
          />
          <Popover
            placement="topLeft"
            trigger="click"
            content={
              <div className="SearchForm--lookbackHint">
                When enabled, search end time is adjusted back by {searchAdjustEndTime} to exclude very
                recent traces that may still be receiving spans.
              </div>
            }
          >
            <IoHelp className="SearchForm--hintTrigger" />
          </Popover>
        </div>
      )}

      <Button
        htmlType="submit"
        className="SearchForm--submit"
        disabled={submitting || noSelectedService || invalid}
      >
        Find Traces
      </Button>
    </Form>
  );

};

export function mapStateToProps(state: ReduxState) {
  const {
    service,
    operation,
    tag: tagParams,
    tags: logfmtTags,
    traceID: traceIDParams,
  } = queryString.parse(state.router.location.search);

  const storedLastSearch = localStorage.getItem('lastSearch');
  const lastSearch = storedLastSearch
    ? (JSON.parse(storedLastSearch) as { service?: string; operation?: string })
    : undefined;
  let lastSearchService: string | undefined;
  let lastSearchOperation: string | undefined;

  if (lastSearch) {
    // last search is only valid if the service is in the list of services
    const { operation: lastOp, service: lastSvc } = lastSearch;
    if (lastSvc && lastSvc !== '-') {
      if (state.services.services && state.services.services.includes(lastSvc)) {
        lastSearchService = lastSvc;
        if (lastOp && lastOp !== '-') {
          const ops = state.services.operationsForService[lastSvc];
          if (lastOp === 'all' || (ops && ops.includes(lastOp))) {
            lastSearchOperation = lastOp;
          }
        }
      }
    }
  }

  let tags: string | undefined;
  // continue to parse tagParams to remain backward compatible with older URLs
  // but, parse to logfmt format instead of the former "key:value|k2:v2"
  if (tagParams) {
    function convFormerTag(accum: Record<string, string>, value: string): boolean {
      const parts = value.split(':', 2);
      const key = parts[0];
      if (key) {
        accum[key] = parts[1] == null ? '' : parts[1];
        return true;
      }
      return false;
    }

    let data: Record<string, string> | null = null;
    if (Array.isArray(tagParams)) {
      data = tagParams
        .filter((str): str is string => !!str) // skip null, undefined, empty strings
        .reduce(
          (accum, str) => {
            convFormerTag(accum, str);
            return accum;
          },
          {} as Record<string, string>
        );
    } else if (typeof tagParams === 'string') {
      const target: Record<string, string> = {};
      data = convFormerTag(target, tagParams) ? target : null;
    }
    if (data) {
      try {
        tags = logfmtStringify(data);
      } catch (_) {
        tags = 'Parse Error';
      }
    } else {
      tags = 'Parse Error';
    }
  }
  if (logfmtTags) {
    let data: Record<string, unknown>;
    try {
      data = JSON.parse(logfmtTags as string);
      tags = logfmtStringify(data);
    } catch (_) {
      tags = 'Parse Error';
    }
  }
  let traceIDs: string | undefined;
  if (traceIDParams) {
    traceIDs = traceIDParams instanceof Array ? traceIDParams.join(',') : (traceIDParams as string);
  }

  return {
    destroyOnUnmount: false,
    initialValues: {
      service: (service as string | undefined) || lastSearchService || '-',
      operation: (operation as string | undefined) || lastSearchOperation || DEFAULT_OPERATION,
      tags,
      traceIDs: traceIDs || null,
    },
    searchAdjustEndTime: _get(state, 'config.search.adjustEndTime'),
    useOtelTerms: _get(state, 'config.useOpenTelemetryTerms'),
  };
}

export function mapDispatchToProps(dispatch: Dispatch) {
  const { searchTraces } = bindActionCreators(jaegerApiActions, dispatch);
  return {
    searchTraces,
    changeServiceHandler: (service: string) =>
      dispatch({
        type: CHANGE_SERVICE_ACTION_TYPE,
        payload: service,
      }),
    submitFormHandler: (
      fields: ISearchFormFields,
      timeRange: TimeRange,
      durationRange: DurationRange,
      resultsLimit: string,
      adjustEndTime: string | null | undefined,
      adjustTimeEnabled: boolean
    ) => submitForm(fields, timeRange, durationRange, resultsLimit, searchTraces, adjustEndTime || null, adjustTimeEnabled),
  };
}

const connector = connect(mapStateToProps, mapDispatchToProps);
type PropsFromRedux = ConnectedProps<typeof connector>;

export default connector(SearchFormImpl);
