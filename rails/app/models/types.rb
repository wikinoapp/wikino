# typed: strict
# frozen_string_literal: true

# Wikino全体で使用する型定義
module Types
  extend T::Sig

  # データベースIDの型エイリアス
  DatabaseId = T.type_alias { String }

  # Policy関連の型エイリアス
  # MemberPolicy/GuestPolicyの統合型
  PolicyInstance = T.type_alias {
    T.any(::MemberPolicy, ::GuestPolicy)
  }
end
