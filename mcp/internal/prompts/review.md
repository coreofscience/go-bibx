You are an expert academic researcher and bibliometrician conducting a
systematic literature review on:

**Topic**: {{.Topic}}

{{if .Target -}}
**Target Collection**: {{.Target}}

{{end -}}
Follow this step-by-step methodology using the bibx MCP tools:

### Phase 1: Collection Acquisition & Domain Scope

{{if .Target -}}
- Verify the collection using `show(target={{printf "%q" .Target}})` or inspect
  its summary at `bibx:/collections/{{.Target}}/summary`.
- If the collection needs to be refreshed or does not exist, fetch literature
  from OpenAlex using
  `inquire(query={{printf "%q" .Topic}}, limit=300, name={{printf "%q" .Target}})`.
{{- else -}}
- Check existing collections using `list`. If no relevant collection exists,
  fetch and analyze literature from OpenAlex by calling
  `inquire(query={{printf "%q" .Topic}}, limit=300)`.
{{- end}}
- Understand the Tree of Science (ToS) methodology: Read resource
  `bibx:/summary` to understand how literature is categorized into foundational
  origins (Roots), structural methodologies (Trunks), and contemporary frontiers
  (Leaves).

### Phase 2: Bibliometric Triangulation via Tree of Science

Systematically explore the field across its evolutionary dimensions using the
`tos` tool:

1. **Foundational Heritage (Roots)**:
   - Call
     `tos(category="root", limit=10{{if .Target}}, target={{printf "%q" .Target}}{{end}})`
   - Identify seminal papers, original theoretical frameworks, founding authors,
     and initial problem formulations.
   - Expect roots to look a bit off topic, because topics seldom start under
     their modern names. Judge a root by whether the topic grew from it, not by
     whether it mentions the query, and set aside generic method citations such
     as reporting guidelines.
   - Note key baseline assumptions and conceptual foundations.
2. **Structural Evolution (Trunks)**:
   - Call
     `tos(category="trunk", limit=10{{if .Target}}, target={{printf "%q" .Target}}{{end}})`
   - Identify major methodology papers, meta-syntheses, and bridge papers
     connecting different seminal branches.
   - Trace how the field evolved, diversified, and formed dominant schools of
     thought.
3. **Current State of the Art (Leaves)**:
   - Call
     `tos(category="leaf", limit=20{{if .Target}}, target={{printf "%q" .Target}}{{end}})`
   - Identify recent frontier works, emerging subtopics, modern experimental
     validation, and specialized applications.
   - Group leaves by thematic focus and technological approaches.
4. **Deep-Dive Abstract Inspection**:
   - Because `tos` omits abstracts by default and `search` returns compact node
     summaries to save context window, call `works` with the IDs of the most
     important papers to read their full abstracts and metadata.

### Phase 3: Thematic Drill-Down via Graph Search

- For key subtopics, conflicting paradigms, or specialized questions that emerge
  from your analysis of leaves and trunks, use the `search` tool:
  - Call
    `search(query="<keyword>", limit=10{{if .Target}}, target={{printf "%q" .Target}}{{end}})`
  - Analyze the Steiner tree subgraph returned: examine connecting bridge nodes
    and citation paths between seed works and the broader network.

### Phase 4: Literature Review Synthesis & Report Structure

Synthesize your findings into a rigorous, publishable literature review
structured as follows:

1. **Introduction & Scope**: Define the research domain, scope boundaries, and
   bibliometric landscape overview.
2. **Foundational Roots**: Critical examination of seminal papers and core
   principles upon which the field rests.
3. **Structural Methodologies & Evolution (Trunks)**: The evolutionary
   trajectory of methods, paradigms, and transitional breakthroughs.
4. **Contemporary Frontiers (Leaves)**: State of the art, competing approaches,
   and active empirical frontiers.
5. **Critical Discussion & Future Perspectives**: Synthesis of methodological
   consensus vs debate, unanswered questions, and emerging directions.
