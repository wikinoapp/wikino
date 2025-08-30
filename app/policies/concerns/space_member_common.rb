# typed: strict
# frozen_string_literal: true

# Spaceメンバー共通のメソッドを定義
module SpaceMemberCommon
  extend T::Sig
  extend T::Helpers

  abstract!

  # Space参加状態の確認
  sig { abstract.returns(T::Boolean) }
  def joined_space?; end

  # 参加しているトピック一覧
  sig { abstract.returns(T.any(TopicRecord::PrivateAssociationRelation, TopicRecord::PrivateRelation)) }
  def joined_topic_records; end

  # 閲覧可能なトピック一覧
  sig { abstract.params(space_record: SpaceRecord).returns(TopicRecord::PrivateAssociationRelation) }
  def showable_topics(space_record:); end

  # 閲覧可能なページ一覧
  sig { abstract.params(space_record: SpaceRecord).returns(PageRecord::PrivateAssociationRelation) }
  def showable_pages(space_record:); end

  # トピック作成権限
  sig { abstract.returns(T::Boolean) }
  def can_create_topic?; end

  # ゴミ箱閲覧権限
  sig { abstract.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_show_trash?(space_record:); end

  # 一括復元権限
  sig { abstract.returns(T::Boolean) }
  def can_create_bulk_restore_pages?; end
end