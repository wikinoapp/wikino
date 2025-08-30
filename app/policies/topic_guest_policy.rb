# typed: strict
# frozen_string_literal: true

# Topic Guest専用のポリシークラス
# ゲスト（非メンバー）が公開トピックにアクセスする際に使用される
# 公開トピックのページ閲覧のみ可能
class TopicGuestPolicy < ApplicationPolicy
  include TopicPermissions

  sig do
    params(
      user_record: T.nilable(UserRecord)
    ).void
  end
  def initialize(user_record:)
    super
  end

  # Topic Guestはトピックの基本情報を更新不可
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_update_topic?(topic_record:)
    false
  end

  # Topic Guestはトピックを削除不可
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_delete_topic?(topic_record:)
    false
  end

  # Topic Guestはトピックメンバーを管理不可
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_manage_topic_members?(topic_record:)
    false
  end

  # Topic Guestはページを作成不可
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_page?(topic_record:)
    false
  end

  # Topic Guestはページを更新不可
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_page?(page_record:)
    false
  end

  # Topic Guestはドラフトページを更新不可
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_draft_page?(page_record:)
    false
  end

  # Topic Guestは公開トピックのページのみ閲覧可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_show_page?(page_record:)
    # 公開トピックのページのみ閲覧可能
    page_record.topic_record.not_nil!.visibility_public?
  end

  # Topic Guestはページをゴミ箱に移動不可
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_trash_page?(page_record:)
    false
  end
end
