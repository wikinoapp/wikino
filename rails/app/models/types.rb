# typed: strict
# frozen_string_literal: true

# Wikino全体で使用する型定義
module Types
  extend T::Sig

  # データベースIDの型エイリアス
  DatabaseId = T.type_alias { String }

  # Policy関連の型エイリアス
  # SpacePolicyFactoryが返す可能性のあるPolicyクラスの型
  SpacePolicyInstance = T.type_alias {
    T.any(::SpaceOwnerPolicy, ::SpaceMemberPolicy, ::SpaceGuestPolicy)
  }

  # Topic層のPolicyクラスの型
  TopicPolicyInstance = T.type_alias {
    T.any(::TopicOwnerPolicy, ::TopicAdminPolicy, ::TopicMemberPolicy, ::TopicGuestPolicy)
  }
end
