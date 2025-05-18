const completions = [
  { label: "[[panic", displayLabel: "panic" },
  { label: "park", type: "constant" },
  { label: "password" },
];

export function wikilinkCompletions(context) {
  let before = context.matchBefore(/\[\[.*/);
  console.log("!!! before:", before);
  console.log("!!! context.explicit:", context.explicit);

  if (!context.explicit && !before) {
    return null;
  }

  const from = before ? before.from : context.pos;
  console.log("!!! from:", from);

  return {
    from,
    options: completions,
    validFor: /^\[\[.*/,
  };
}
