export class Session {
  static active: Session | undefined;

  activate() {
    if (Session.active !== undefined && Session.active !== this) {
      console.warn("Another session is active: neither stays active 8333");
      Session.active = undefined;
    }
    Session.active = this;
    return this;
  }
}
