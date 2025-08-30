# typed: strict
# frozen_string_literal: true

module T
  module Wikino
    extend T::Sig

    DatabaseId = T.type_alias { String }

    # Policy関連の型エイリアス
    # SpacePolicyFactoryが返す可能性のあるPolicyクラスの型
    SpacePolicyInstance = T.type_alias { T.any(SpaceOwnerPolicy, SpaceMemberPolicy, SpaceGuestPolicy) }

    # Topic層のPolicyクラスの型
    TopicPolicyInstance = T.type_alias { T.any(TopicAdminPolicy, TopicMemberPolicy) }
  end
end
