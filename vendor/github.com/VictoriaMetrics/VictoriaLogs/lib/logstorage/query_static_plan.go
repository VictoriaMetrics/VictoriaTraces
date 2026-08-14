package logstorage

// QueryStaticPlan is the execution-independent portion of a query plan.
type QueryStaticPlan struct {
	OptimizedQuery       string   `json:"optimized_query"`
	RemoteQuery          string   `json:"remote_query"`
	RemotePipeNames      []string `json:"remote_pipe_names"`
	LocalPipeNames       []string `json:"local_pipe_names"`
	UnresolvedSubqueries []string `json:"unresolved_subqueries"`
}

// BuildQueryStaticPlan parses query at timestamp and returns the portions of its optimized
// distributed plan that can be determined without executing subqueries or accessing storage.
func BuildQueryStaticPlan(query string, timestamp int64) (QueryStaticPlan, error) {
	q, err := ParseQueryAtTimestamp(query, timestamp)
	if err != nil {
		return QueryStaticPlan{}, err
	}
	return GetQueryStaticPlan(q), nil
}

// GetQueryStaticPlan returns the portions of q's optimized distributed plan that can be
// determined without executing subqueries or accessing storage.
func GetQueryStaticPlan(q *Query) QueryStaticPlan {
	qRemote, pipesLocal := splitQueryToRemoteAndLocal(q)
	remotePipeNames := make([]string, len(qRemote.pipes))
	for i, p := range qRemote.pipes {
		remotePipeNames[i] = getPipeName(p)
	}
	localPipeNames := make([]string, len(pipesLocal))
	for i, p := range pipesLocal {
		localPipeNames[i] = getPipeName(p)
	}

	var unresolvedSubqueries []string
	q.visitSubqueries(func(qSubquery *Query) {
		if qSubquery != q {
			unresolvedSubqueries = append(unresolvedSubqueries, qSubquery.String())
		}
	})
	if remotePipeNames == nil {
		remotePipeNames = []string{}
	}
	if localPipeNames == nil {
		localPipeNames = []string{}
	}
	if unresolvedSubqueries == nil {
		unresolvedSubqueries = []string{}
	}

	return QueryStaticPlan{
		OptimizedQuery:       q.String(),
		RemoteQuery:          qRemote.String(),
		RemotePipeNames:      remotePipeNames,
		LocalPipeNames:       localPipeNames,
		UnresolvedSubqueries: unresolvedSubqueries,
	}
}
