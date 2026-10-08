let pathname = "/";
let params = {};
const pushes = [];

export function usePathname() {
  return pathname;
}

export function useRouter() {
  return {
    push(to) {
      pushes.push(to);
    },
    replace(to) {
      pushes.push(to);
    },
    refresh() {},
  };
}

export function useParams() {
  return params;
}

export function __setPathname(value) {
  pathname = value;
}

export function __setParams(value) {
  params = value || {};
}

export function __takePushes() {
  return pushes.splice(0, pushes.length);
}
