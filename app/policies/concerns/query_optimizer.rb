# typed: strict
# frozen_string_literal: true

# データベースクエリ最適化のためのヘルパーモジュール
module QueryOptimizer
  extend T::Sig
  extend T::Helpers

  # N+1クエリを防ぐためのプリロードヘルパー
  sig { params(space_member_record: T.nilable(SpaceMemberRecord)).returns(T.nilable(SpaceMemberRecord)) }
  def preload_space_member_associations(space_member_record)
    return nil if space_member_record.nil?

    # 権限チェックでよく使用される関連をプリロード
    SpaceMemberRecord
      .preload(:space_record, :user_record, :topic_member_records)
      .find_by(id: space_member_record.id)
  end

  sig { params(topic_member_record: T.nilable(TopicMemberRecord)).returns(T.nilable(TopicMemberRecord)) }
  def preload_topic_member_associations(topic_member_record)
    return nil if topic_member_record.nil?

    # Topic権限チェックでよく使用される関連をプリロード
    TopicMemberRecord
      .preload(:topic_record, :space_member_record)
      .find_by(id: topic_member_record.id)
  end

  # ページの権限チェックに必要な関連をプリロード
  sig { params(page_record: PageRecord).returns(PageRecord) }
  def preload_page_associations(page_record)
    PageRecord
      .preload(topic_record: [:space_record, :member_records])
      .find(page_record.id)
  end

  # トピックの権限チェックに必要な関連をプリロード
  sig { params(topic_record: TopicRecord).returns(TopicRecord) }
  def preload_topic_associations(topic_record)
    TopicRecord
      .preload(:space_record, :member_records)
      .find(topic_record.id)
  end

  # バッチでの権限チェック用
  sig { params(page_records: T::Array[PageRecord]).returns(T::Array[PageRecord]) }
  def preload_pages_for_permission_check(page_records)
    return [] if page_records.empty?

    PageRecord
      .where(id: page_records.map(&:id))
      .preload(
        :space_record,
        topic_record: [:space_record, :member_records]
      )
      .to_a
  end
end

