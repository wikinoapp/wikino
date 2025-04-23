# typed: strict
# frozen_string_literal: true

class Page < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  # ページをゴミ箱に移動してから削除されるまでの日数
  DELETE_LIMIT_DAYS = 30
  # タイトルの最大文字数 (値に強い理由は無い)
  TITLE_MAX_LENGTH = 200

  const :database_id, T::Wikino::DatabaseId
  const :number, Integer
  const :title, T.nilable(String)
  const :body, String
  const :body_html, String
  const :modified_at, ActiveSupport::TimeWithZone
  const :published_at, T.nilable(ActiveSupport::TimeWithZone)
  const :pinned_at, T.nilable(ActiveSupport::TimeWithZone)
  const :space, Space
  const :topic, Topic

  sig { returns(T::Boolean) }
  def published?
    published_at.present?
  end

  sig { returns(T::Boolean) }
  def pinned?
    pinned_at.present?
  end

  sig { returns(T::Boolean) }
  def modified_after_published?
    published? && modified_at > published_at
  end
end
