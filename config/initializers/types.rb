# typed: strict
# frozen_string_literal: true

module T
  module Wikino
    extend T::Sig

    DatabaseId = T.type_alias { String }

    # Policy関連の型エイリアス
    # SpaceMemberPolicyFactoryとPermissionResolverが返す可能性のあるPolicyクラスの共用型
    PolicyInstance = T.type_alias { T.any(SpaceOwnerPolicy, SpaceMemberPolicy, SpaceGuestPolicy, TopicAdminPolicy) }
  end
end
