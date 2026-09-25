import AccountView from "./view";

export default async function AccountPage({ params }) {
  const { name } = await params;
  return <AccountView name={name} />;
}
