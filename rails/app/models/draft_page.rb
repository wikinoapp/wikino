# typed: strict
# frozen_string_literal: true

class DraftPage < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :database_id, Types::DatabaseId
  const :title, T.nilable(String)
  const :modified_at, ActiveSupport::TimeWithZone
  const :space, Space
  const :page, Page

  sig { returns(String) }
  def display_title
    title.presence || page.title.presence || I18n.t("messages.pages.untitled")
  end
end
