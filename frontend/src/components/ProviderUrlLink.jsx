export default function ProviderUrlLink({ url, className = "" }) {
  if (!url) return null;

  const classes = ["provider-url-link", className].filter(Boolean).join(" ");
  return (
    <a className={classes} href={url} target="_blank" rel="noopener noreferrer">
      <code>{url}</code>
    </a>
  );
}
