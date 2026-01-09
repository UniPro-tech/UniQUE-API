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

impl Permission {
    /// Create a single-flag Permission from its string name.
    pub fn from_str(s: &str) -> Option<Permission> {
        match s {
            PermissionString::USER_READ => Some(Permission::USER_READ),
            PermissionString::USER_CREATE => Some(Permission::USER_CREATE),
            PermissionString::USER_UPDATE => Some(Permission::USER_UPDATE),
            PermissionString::USER_DELETE => Some(Permission::USER_DELETE),
            PermissionString::USER_DISABLE => Some(Permission::USER_DISABLE),

            PermissionString::APP_READ => Some(Permission::APP_READ),
            PermissionString::APP_CREATE => Some(Permission::APP_CREATE),
            PermissionString::APP_UPDATE => Some(Permission::APP_UPDATE),
            PermissionString::APP_DELETE => Some(Permission::APP_DELETE),
            PermissionString::APP_SECRET_ROTATE => Some(Permission::APP_SECRET_ROTATE),

            PermissionString::TOKEN_REVOKE => Some(Permission::TOKEN_REVOKE),
            PermissionString::AUDIT_READ => Some(Permission::AUDIT_READ),
            PermissionString::CONFIG_UPDATE => Some(Permission::CONFIG_UPDATE),
            PermissionString::KEY_MANAGE => Some(Permission::KEY_MANAGE),

            PermissionString::ROLE_MANAGE => Some(Permission::ROLE_MANAGE),
            PermissionString::PERMISSION_MANAGE => Some(Permission::PERMISSION_MANAGE),
            PermissionString::SESSION_MANAGE => Some(Permission::SESSION_MANAGE),
            PermissionString::MFA_MANAGE => Some(Permission::MFA_MANAGE),

            _ => None,
        }
    }

    /// Check whether this Permission contains the flag represented by the given string.
    pub fn contains_str(&self, s: &str) -> bool {
        if let Some(p) = Permission::from_str(s) {
            self.contains(p)
        } else {
            false
        }
    }

    /// Given a raw bits value, return a tuple of (known_names, known_mask).
    /// `known_names` is a Vec<String> of matching `PermissionString` names.
    /// `known_mask` is the bitmask of all recognized bits.
    pub fn names_from_bits(bits: u32) -> (Vec<String>, u32) {
        const KNOWN: &[(u32, &str)] = &[
            (Permission::USER_READ.bits(), PermissionString::USER_READ),
            (
                Permission::USER_CREATE.bits(),
                PermissionString::USER_CREATE,
            ),
            (
                Permission::USER_UPDATE.bits(),
                PermissionString::USER_UPDATE,
            ),
            (
                Permission::USER_DELETE.bits(),
                PermissionString::USER_DELETE,
            ),
            (
                Permission::USER_DISABLE.bits(),
                PermissionString::USER_DISABLE,
            ),
            (Permission::APP_READ.bits(), PermissionString::APP_READ),
            (Permission::APP_CREATE.bits(), PermissionString::APP_CREATE),
            (Permission::APP_UPDATE.bits(), PermissionString::APP_UPDATE),
            (Permission::APP_DELETE.bits(), PermissionString::APP_DELETE),
            (
                Permission::APP_SECRET_ROTATE.bits(),
                PermissionString::APP_SECRET_ROTATE,
            ),
            (
                Permission::TOKEN_REVOKE.bits(),
                PermissionString::TOKEN_REVOKE,
            ),
            (Permission::AUDIT_READ.bits(), PermissionString::AUDIT_READ),
            (
                Permission::CONFIG_UPDATE.bits(),
                PermissionString::CONFIG_UPDATE,
            ),
            (Permission::KEY_MANAGE.bits(), PermissionString::KEY_MANAGE),
            (
                Permission::ROLE_MANAGE.bits(),
                PermissionString::ROLE_MANAGE,
            ),
            (
                Permission::PERMISSION_MANAGE.bits(),
                PermissionString::PERMISSION_MANAGE,
            ),
            (
                Permission::SESSION_MANAGE.bits(),
                PermissionString::SESSION_MANAGE,
            ),
            (Permission::MFA_MANAGE.bits(), PermissionString::MFA_MANAGE),
        ];

        let mut res = Vec::new();
        let mut mask = 0u32;
        for &(b, name) in KNOWN.iter() {
            if (bits & b) != 0 {
                res.push(name.to_string());
                mask |= b;
            }
        }
        (res, mask)
    }
}

pub struct PermissionString;

impl PermissionString {
    pub const USER_READ: &'static str = "USER_READ";
    pub const USER_CREATE: &'static str = "USER_CREATE";
    pub const USER_UPDATE: &'static str = "USER_UPDATE";
    pub const USER_DELETE: &'static str = "USER_DELETE";
    pub const USER_DISABLE: &'static str = "USER_DISABLE";

    pub const APP_READ: &'static str = "APP_READ";
    pub const APP_CREATE: &'static str = "APP_CREATE";
    pub const APP_UPDATE: &'static str = "APP_UPDATE";
    pub const APP_DELETE: &'static str = "APP_DELETE";
    pub const APP_SECRET_ROTATE: &'static str = "APP_SECRET_ROTATE";

    pub const TOKEN_REVOKE: &'static str = "TOKEN_REVOKE";
    pub const AUDIT_READ: &'static str = "AUDIT_READ";
    pub const CONFIG_UPDATE: &'static str = "CONFIG_UPDATE";
    pub const KEY_MANAGE: &'static str = "KEY_MANAGE";

    pub const ROLE_MANAGE: &'static str = "ROLE_MANAGE";
    pub const PERMISSION_MANAGE: &'static str = "PERMISSION_MANAGE";
    pub const SESSION_MANAGE: &'static str = "SESSION_MANAGE";
    pub const MFA_MANAGE: &'static str = "MFA_MANAGE";
}
