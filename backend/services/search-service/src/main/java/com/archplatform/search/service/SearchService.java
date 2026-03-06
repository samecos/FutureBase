package com.archplatform.search.service;

import co.elastic.clients.elasticsearch.ElasticsearchClient;
import co.elastic.clients.elasticsearch._types.SortOrder;
import co.elastic.clients.elasticsearch.core.SearchRequest;
import co.elastic.clients.elasticsearch.core.SearchResponse;
import co.elastic.clients.elasticsearch.core.search.HighlightField;
import co.elastic.clients.elasticsearch.core.search.Hit;
import com.archplatform.search.dto.SearchRequest;
import com.archplatform.search.dto.SearchResponse.SearchHit;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;

import java.io.IOException;
import java.util.*;
import java.util.stream.Collectors;

@Slf4j
@Service
@RequiredArgsConstructor
public class SearchService {

    private final ElasticsearchClient elasticsearchClient;

    public com.archplatform.search.dto.SearchResponse search(SearchRequest request) throws IOException {
        List<String> indices = request.getIndices() != null ? request.getIndices() : List.of("projects", "designs", "elements");
        
        // Build the search query
        var boolQuery = co.elastic.clients.elasticsearch._types.QueryBuilders.bool()
            .must(m -> m.multiMatch(mm -> mm
                .query(request.getQuery())
                .fields("name^3", "description^2", "tags", "content")
                .fuzziness("AUTO")
            ));

        // Add tenant filter
        if (request.getTenantId() != null) {
            boolQuery.filter(f -> f.term(t -> t.field("tenantId").value(request.getTenantId().toString())));
        }

        // Add project filter
        if (request.getProjectId() != null) {
            boolQuery.filter(f -> f.term(t -> t.field("projectId").value(request.getProjectId().toString())));
        }

        // Add custom filters
        if (request.getFilters() != null) {
            request.getFilters().forEach((key, value) -> {
                boolQuery.filter(f -> f.term(t -> t.field(key).value(value)));
            });
        }

        // Build highlight
        Map<String, HighlightField> highlightFields = new HashMap<>();
        highlightFields.put("name", HighlightField.of(h -> h));
        highlightFields.put("description", HighlightField.of(h -> h));

        // Build search request
        SearchRequest.Builder searchBuilder = new SearchRequest.Builder()
            .index(indices)
            .query(q -> q.bool(boolQuery.build()))
            .from(request.getPage() * request.getSize())
            .size(request.getSize())
            .trackTotalHits(t -> t.enabled(true));

        // Add highlighting
        if (request.getHighlight() != null && request.getHighlight()) {
            searchBuilder.highlight(h -> h
                .fields(highlightFields)
                .preTags("<mark>")
                .postTags("</mark>")
                .fragmentSize(150)
                .numberOfFragments(3)
            );
        }

        // Add sorting
        String sortBy = request.getSortBy() != null ? request.getSortBy() : "_score";
        SortOrder sortOrder = "desc".equalsIgnoreCase(request.getSortOrder()) ? SortOrder.Desc : SortOrder.Asc;
        searchBuilder.sort(s -> s.field(f -> f.field(sortBy).order(sortOrder)));

        SearchResponse<Map> response = elasticsearchClient.search(searchBuilder.build(), Map.class);

        // Map results
        List<SearchHit> hits = response.hits().hits().stream()
            .map(this::mapToSearchHit)
            .collect(Collectors.toList());

        long totalHits = response.hits().total() != null ? response.hits().total().value() : 0;
        int totalPages = (int) Math.ceil((double) totalHits / request.getSize());

        return com.archplatform.search.dto.SearchResponse.builder()
            .hits(hits)
            .totalHits(totalHits)
            .totalPages(totalPages)
            .currentPage(request.getPage())
            .took(response.took())
            .timedOut(response.timedOut())
            .build();
    }

    public List<String> getSuggestions(String query, String index) throws IOException {
        if (query == null || query.length() < 2) {
            return List.of();
        }

        var suggestRequest = co.elastic.clients.elasticsearch.core.SuggestRequest.of(sr -> sr
            .index(index != null ? index : "projects")
            .suggest("name-suggest", s -> s
                .completion(c -> c
                    .field("suggest")
                    .prefix(query)
                    .size(10)
                )
            )
        );

        var response = elasticsearchClient.suggest(suggestRequest);
        
        return response.suggest().get("name-suggest").stream()
            .flatMap(s -> s.completion().options().stream())
            .map(o -> o.text())
            .distinct()
            .limit(10)
            .collect(Collectors.toList());
    }

    public Map<String, Long> getAggregations(String field, String index) throws IOException {
        var aggRequest = SearchRequest.of(sr -> sr
            .index(index != null ? index : "projects")
            .size(0)
            .aggregations("terms-agg", a -> a
                .terms(t -> t.field(field))
            )
        );

        var response = elasticsearchClient.search(aggRequest, Map.class);
        
        var termsAgg = response.aggregations().get("terms-agg");
        Map<String, Long> result = new HashMap<>();
        
        if (termsAgg != null && termsAgg.sterms() != null) {
            termsAgg.sterms().buckets().array().forEach(bucket -> {
                result.put(bucket.key().stringValue(), bucket.docCount());
            });
        }
        
        return result;
    }

    private SearchHit mapToSearchHit(Hit<Map> hit) {
        Map<String, Object> source = hit.source();
        
        Map<String, List<String>> highlights = new HashMap<>();
        if (hit.highlight() != null) {
            hit.highlight().forEach((key, values) -> highlights.put(key, values));
        }

        return SearchHit.builder()
            .id(hit.id())
            .index(hit.index())
            .type(hit.index())
            .title(source != null ? (String) source.get("name") : null)
            .description(source != null ? (String) source.get("description") : null)
            .source(source)
            .highlights(highlights)
            .score(hit.score())
            .build();
    }
}
