# typed: strict
# frozen_string_literal: true

# 権限チェックの基底クラス
# SpaceメンバーのPolicyクラスで共通して使用するロジックを提供
class BaseMemberPolicy < ApplicationPolicy
  sig do
    params(
      user_record: T.nilable(UserRecord),
      space_member_record: T.nilable(SpaceMemberRecord)
    ).void
  end
  def initialize(user_record:, space_member_record:)
    @user_record = user_record
    @space_member_record = space_member_record

    # user_recordとspace_member_recordの関連性を検証
    if mismatched_relations?
      raise ArgumentError, [
        "Mismatched relations.",
        "user_record.id: #{user_record&.id.inspect}",
        "space_member_record.user_id: #{space_member_record&.user_id.inspect}"
      ].join(" ")
    end
  end

  # スペースに参加しているかどうか
  sig { returns(T::Boolean) }
  def joined_space?
    !space_member_record.nil?
  end

  # 指定されたスペースIDと同じスペースにいるかどうか
  sig { params(space_id: String).returns(T::Boolean) }
  def in_same_space?(space_id:)
    space_member_record&.space_id == space_id
  end

  # アクティブなメンバーかどうか
  sig { returns(T::Boolean) }
  def active?
    space_member_record&.active? || false
  end

  # 参加しているトピックのレコードを取得
  sig { returns(T.nilable(::ActiveRecord::Relation)) }
  def joined_topic_records
    return nil if space_member_record.nil?

    space_member_record!.topic_records
  end

  # トピックに参加しているかどうか
  sig { params(topic_id: String).returns(T::Boolean) }
  def joined_topic?(topic_id:)
    return false if space_member_record.nil?

    space_member_record!.topic_records.where(id: topic_id).exists?
  end

  protected

  sig { returns(T.nilable(UserRecord)) }
  attr_reader :user_record

  sig { returns(T.nilable(SpaceMemberRecord)) }
  attr_reader :space_member_record

  # space_member_record の non-nil バージョン
  sig { returns(SpaceMemberRecord) }
  def space_member_record!
    T.must(space_member_record)
  end

  private

  # user_recordとspace_member_recordの関連性が不整合かどうか
  sig { returns(T::Boolean) }
  def mismatched_relations?
    return false if user_record.nil? || space_member_record.nil?

    T.must(user_record).id != T.must(space_member_record).user_id
  end
end
