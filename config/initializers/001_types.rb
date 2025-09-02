# typed: strict
# frozen_string_literal: true

# 型定義を早期に読み込むためのイニシャライザー
# ファイル名を001_で始めることで、他のイニシャライザーより先に読み込まれる

module T
  module Wikino
    extend T::Sig

    DatabaseId = T.type_alias { String }

    # Policy関連の型エイリアス
    # SpacePolicyFactoryが返す可能性のあるPolicyクラスの型
    SpacePolicyInstance = T.type_alias { T.any(SpaceOwnerPolicy, SpaceMemberPolicy, SpaceGuestPolicy) }

    # Topic層のPolicyクラスの型
    TopicPolicyInstance = T.type_alias { T.any(TopicOwnerPolicy, TopicAdminPolicy, TopicMemberPolicy, TopicGuestPolicy) }
  end
end