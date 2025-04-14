# typed: strict
# frozen_string_literal: true

class SpaceRecord < ApplicationRecord
  include Discard::Model

  include FormConcerns::ISpace

  IDENTIFIER_FORMAT = /\A[A-Za-z0-9-]+\z/
  # 識別子の最大文字数 (値に強い理由は無い)
  IDENTIFIER_MAX_LENGTH = 20
  # 識別子の予約語
  IDENTIFIER_RESERVED_WORDS = %w[www].freeze
  # 名前の最大文字数 (値に強い理由は無い)
  NAME_MAX_LENGTH = 30

  self.table_name = "spaces"

  enum :plan, {
    Plan::Free.serialize => 0,
    Plan::Small.serialize => 1,
    Plan::Large.serialize => 2
  }, prefix: true

  has_many :draft_pages, dependent: :restrict_with_exception
  has_many :exports, dependent: :restrict_with_exception
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

  sig { params(identifier: String).returns(Space) }
  def self.find_by_identifier!(identifier)
    kept.find_by!(identifier:)
  end

  sig { override.params(identifier: String).returns(T::Boolean) }
  def identifier_uniqueness?(identifier)
    Space.where.not(id:).exists?(identifier:)
  end

  sig { params(space_viewer: ModelConcerns::SpaceViewable).returns(SpaceEntity) }
  def to_model(space_viewer:)
    Space.new(
      database_id: id,
      identifier:,
      name:,
      plan: Plan.deserialize(plan),
      joined_at:,
      viewer_can_update: space_viewer.can_update_space?(space: self),
      viewer_can_export: space_viewer.can_export_space?(space: self)
    )
  end

  sig { params(number: Integer).returns(Page) }
  def find_page_by_number!(number)
    pages.kept.find_by!(number:)
  end

  sig do
    params(
      space_viewer: ModelConcerns::SpaceViewable,
      before: T.nilable(String),
      after: T.nilable(String)
    ).returns(PageListEntity)
  end
  def restorable_page_list_entity(space_viewer:, before:, after:)
    cursor_paginate_page = pages.preload(:topic).restorable.cursor_paginate(
      before: before.presence,
      after: after.presence,
      limit: 100,
      order: {trashed_at: :desc, id: :desc}
    ).fetch
    page_entities = Page.to_entities(space_viewer:, pages: cursor_paginate_page.records)
    pagination_entity = PaginationEntity.from_cursor_paginate(cursor_paginate_page:)

    PageListEntity.new(page_entities:, pagination_entity:)
  end

  sig { params(user: User, role: SpaceMemberRole, joined_at: ActiveSupport::TimeWithZone).returns(T.untyped) }
  def add_member!(user:, role:, joined_at:)
    space_members.create!(user:, role: role.serialize, joined_at:)
  end
end
