# typed: strict
# frozen_string_literal: true

class Space < ApplicationRecord
  include Discard::Model

  IDENTIFIER_FORMAT = /\A[A-Za-z0-9-]+\z/
  IDENTIFIER_MIN_LENGTH = 2
  IDENTIFIER_MAX_LENGTH = 20
  RESERVED_IDENTIFIERS = %w[www].freeze

  enum :plan, {
    Plan::Free.serialize => 0,
    Plan::Small.serialize => 1,
    Plan::Large.serialize => 2
  }, prefix: true

  has_many :draft_pages, dependent: :restrict_with_exception
  has_many :topic_members, dependent: :restrict_with_exception
  has_many :topics, dependent: :restrict_with_exception
  has_many :page_editors, dependent: :restrict_with_exception
  has_many :page_revisions, dependent: :restrict_with_exception
  has_many :pages, dependent: :restrict_with_exception
  has_many :space_members, dependent: :restrict_with_exception
  has_many :users, dependent: :restrict_with_exception

  sig do
    params(identifier: String, current_time: ActiveSupport::TimeWithZone, locale: ViewerLocale)
      .returns(Space)
  end
  def self.create_initial_space!(identifier:, current_time:, locale:)
    create!(
      identifier:,
      plan: Plan::Small.serialize,
      name: I18n.t("messages.spaces.new_space", locale: locale.serialize),
      joined_at: current_time
    )
  end

  sig { params(number: Integer).returns(Page) }
  def find_page_by_number!(number)
    pages.kept.find_by!(number:)
  end

  sig { params(before: T.nilable(String), after: T.nilable(String)).returns(PageConnection) }
  def restorable_page_connection(before:, after:)
    cursor_paginate_page = pages.preload(:topic).restorable.cursor_paginate(
      before: before.presence,
      after: after.presence,
      limit: 100,
      order: {trashed_at: :desc, id: :desc}
    ).fetch

    PageConnection.new(
      pages: cursor_paginate_page.records,
      pagination: Pagination.from_cursor_paginate(cursor_paginate_page:)
    )
  end
end
