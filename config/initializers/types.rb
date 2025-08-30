# typed: strict
# frozen_string_literal: true

module T
  module Wikino
    extend T::Sig

    DatabaseId = T.type_alias { String }

    # Policy関連の型エイリアス
    # SpaceMemberPolicyFactoryが返す可能性のあるPolicyクラスの型
    SpacePolicyInstance = T.type_alias { T.any(SpaceOwnerPolicy, SpaceMemberPolicy, SpaceGuestPolicy) }

    # Topic層のPolicyクラスの型
    TopicPolicyInstance = T.type_alias { T.any(TopicAdminPolicy, TopicMemberPolicy) }

    # PermissionResolverが返す可能性のあるPolicyクラスの型（Topic権限を含む）
    PolicyInstance = T.type_alias { T.any(SpaceOwnerPolicy, SpaceMemberPolicy, SpaceGuestPolicy, TopicAdminPolicy) }
  end
end
