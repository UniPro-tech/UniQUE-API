use bitflags::bitflags;

bitflags! {
  /// ## Permissions
  /// 権限ビットフラグの定義
  /// 各ビットは特定の操作に対応し、組み合わせて使用することで複数の権限を表現できる.
  /// 例えば、USER_READ | USER_CREATE はユーザーの読み取りと作成の両方の権限を持つことを意味する.
  /// ## 設計方針
  /// 0-7     = USER 系
  /// 8-15    = CLIENT 系
  /// 16-23   = SYSTEM 系
  /// 24-31   = 予備
  #[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
  pub struct Permission: u32 {
    // --- User Management ---
    // NOTE: もちろん自身のユーザー情報の更新・削除は別途許可される.
    // ここでは他者のユーザー情報に対する操作権限を定義する.
    const USER_READ       = 1 << 0; // ユーザー読み取り
    const USER_CREATE     = 1 << 1; // ユーザー作成
    const USER_UPDATE     = 1 << 2; // ユーザー更新(パスワード無効化も含まれる)
    const USER_DELETE     = 1 << 3; // ユーザー削除
    const USER_DISABLE    = 1 << 4; // ユーザー無効化

    // --- Apps Management ---
    // NOTE: App は OAuth2 Client に相当する.
    // また、自身がOwnerである App に対しては別途許可される.
    // ここでは他者が所有する App に対する操作権限を定義する.
    const APP_READ     = 1 << 8; // App読み取り
    const APP_CREATE   = 1 << 9; // App作成
    const APP_UPDATE   = 1 << 10; // App更新
    const APP_DELETE   = 1 << 11; // App削除
    const APP_SECRET_ROTATE = 1 << 12; // Appシークレットの再発行

    // --- System / Config ---
    const TOKEN_REVOKE    = 1 << 16; // トークンの取り消し
    const AUDIT_READ      = 1 << 18; // 監査ログの読み取り
    const CONFIG_UPDATE   = 1 << 19; //　 全体設定（認証フロー、署名鍵など）の変更
    const KEY_MANAGE      = 1 << 20; // JWK鍵管理（追加／削除）

    // --- RBAC / Security ---
    const ROLE_MANAGE     = 1 << 24; // RBACロール自体の作成／編集／削除
    const PERMISSION_MANAGE = 1 << 25; // RBAC権限の割り当て管理
    const SESSION_MANAGE  = 1 << 26; // セッション管理（強制ログアウトなど）
    const MFA_MANAGE      = 1 << 27; // 多要素認証の管理(リセットなど)
  }
}
