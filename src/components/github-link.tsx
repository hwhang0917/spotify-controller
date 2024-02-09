import { GitHubLogoIcon } from "@radix-ui/react-icons";
import { Button } from "@/components/ui/button";
import Link from "next/link";

interface GithubLinkProps {
  href: string;
}

export function GithubLink(props: GithubLinkProps) {
  return (
    <Button variant="outline" size="icon">
      <Link href={props.href} target="_blank">
        <GitHubLogoIcon className="h-[1.2rem] w-[1.2rem]" />
        <span className="sr-only">GitHub Repository Link</span>
      </Link>
    </Button>
  );
}
