export async function wikilinkCompletions(spaceIdentifier: string) {
  return async (context) => {
    let before = context.matchBefore(/\[\[.*/);

    if (!context.explicit && !before) {
      return null;
    }

    const from = before ? before.from : context.pos;
    const completions = await buildCompletions(spaceIdentifier, before);

    return {
      from,
      options: completions,
      // `foo bar` というタイトルのページがある状態で、
      // `[[bar foo` と入力したとき補完候補に出るようにするためにフィルタを無効化している
      filter: false,
    };
  };
}

async function fetchPageLocations(spaceIdentifier: string, keyword: string) {
  return fetch(`/s/${spaceIdentifier}/page_locations?q=${keyword}`)
    .then((response) => response.json())
    .then((data) => data.page_locations);
}

async function buildCompletions(spaceIdentifier: string, before: any) {
  if (!before) {
    return [];
  }

  // [[foo/bar の [[foo/ を取り除き bar を取得する
  const keyword = before.text.replace(/^\[\[/, "").replace(/.*\//, "");

  const pageLocations = await fetchPageLocations(spaceIdentifier, keyword);

  return pageLocations.map((pageLocation) => ({
    label: `[[${pageLocation.key}`,
    displayLabel: pageLocation.key,
  }));
}
