export type SecretLinkParts = {
  fullLink: string;
  publicUrl: string;
  fragmentKey: string;
};

export function splitSecretLink(secretLink: string): SecretLinkParts {
  const fullLink = secretLink.trim();
  if (!fullLink) {
    return {
      fullLink: "",
      publicUrl: "",
      fragmentKey: "",
    };
  }

  const hashIndex = fullLink.indexOf("#");
  if (hashIndex === -1) {
    return {
      fullLink,
      publicUrl: fullLink,
      fragmentKey: "",
    };
  }

  return {
    fullLink,
    publicUrl: fullLink.slice(0, hashIndex),
    fragmentKey: fullLink.slice(hashIndex + 1),
  };
}
