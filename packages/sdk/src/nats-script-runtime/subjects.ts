const placeholderPattern = /<[^>]+>|\{[^}]+\}/;

export function hasPlaceholderSubject(subject: string): boolean {
  return placeholderPattern.test(subject);
}

export function isValidSubjectPattern(subject: string): boolean {
  const tokens = subject.split(".");
  if (tokens.length === 0 || tokens.some((token) => token.length === 0)) {
    return false;
  }

  for (let index = 0; index < tokens.length; index += 1) {
    const token = tokens[index];
    const isWildcard = token === "*" || token === ">";

    if ((token.includes("*") || token.includes(">")) && !isWildcard) {
      return false;
    }
    if (token === ">" && index !== tokens.length - 1) {
      return false;
    }
    if (isWildcard && index < 2) {
      return false;
    }
  }

  return true;
}

export function matchSubject(pattern: string, subject: string): boolean {
  const patternTokens = pattern.split(".");
  const subjectTokens = subject.split(".");

  for (
    let patternIndex = 0, subjectIndex = 0;
    patternIndex < patternTokens.length;
    patternIndex += 1, subjectIndex += 1
  ) {
    const token = patternTokens[patternIndex];

    if (token === ">") {
      return subjectTokens.length > subjectIndex;
    }
    if (subjectIndex >= subjectTokens.length) {
      return false;
    }
    if (token !== "*" && token !== subjectTokens[subjectIndex]) {
      return false;
    }
  }

  return patternTokens.length === subjectTokens.length;
}
