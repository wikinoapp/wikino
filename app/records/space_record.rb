# typed: strict
# frozen_string_literal: true

class SpaceRecord < ApplicationRecord
  include Discard::Model

  include FormConcerns::ISpace

  self.table_name = "spaces"

  enum :plan, {
    Plan::Free.serialize => 0,
    Plan::Small.serialize => 1,
    Plan::Large.serialize => 2
  }, prefix: true

  has_many :draft_page_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_id,
    inverse_of: :space_record
  has_many :export_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_id,
    inverse_of: :space_record
  has_many :topic_member_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_id,
    inverse_of: :space_record
  has_many :topic_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_id,
    inverse_of: :space_record
  has_many :page_editor_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_id,
    inverse_of: :space_record
  has_many :page_revision_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_id,
    inverse_of: :space_record
  has_many :page_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_id,
    inverse_of: :space_record
  has_many :space_member_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_id,
    inverse_of: :space_record

  sig do
    params(identifier: String, current_time: ActiveSupport::TimeWithZone, locale: ViewerLocale)
      .returns(SpaceRecord)
  end
  def self.create_initial_space!(identifier:, current_time:, locale:)
    create!(
      identifier:,
      plan: Plan::Small.serialize,
      name: I18n.t("messages.spaces.new_space", locale: locale.serialize),
      joined_at: current_time
    )
  end

  sig { params(identifier: String).returns(SpaceRecord) }
  def self.find_by_identifier!(identifier)
    kept.find_by!(identifier:)
  end

  sig { override.params(identifier: String).returns(T::Boolean) }
  def identifier_uniqueness?(identifier)
    SpaceRecord.where.not(id:).exists?(identifier:)
  end

  sig { params(space_viewer: ModelConcerns::SpaceViewable).returns(SpaceEntity) }
  def to_entity(space_viewer:)
    SpaceEntity.new(
      database_id: id,
      identifier:,
      name:,
      plan: Plan.deserialize(plan),
      joined_at:,
      viewer_can_update: space_viewer.can_update_space?(space: self),
      viewer_can_export: space_viewer.can_export_space?(space: self)
    )
  end

  sig { params(number: Integer).returns(PageRecord) }
  def find_page_by_number!(number)
    page_records.kept.find_by!(number:)
  end

  sig do
    params(
      space_viewer: ModelConcerns::SpaceViewable,
      before: T.nilable(String),
      after: T.nilable(String)
    ).returns(PageListEntity)
  end
  def restorable_page_list_entity(space_viewer:, before:, after:)
    cursor_paginate_page = page_records.preload(:topic_record).restorable.cursor_paginate(
      before: before.presence,
      after: after.presence,
      limit: 100,
      order: {trashed_at: :desc, id: :desc}
    ).fetch
    page_entities = PageRecord.to_entities(space_viewer:, pages: cursor_paginate_page.records)
    pagination_entity = PaginationEntity.from_cursor_paginate(cursor_paginate_page:)

    PageListEntity.new(page_entities:, pagination_entity:)
  end

  sig { params(user: UserRecord, role: SpaceMemberRole, joined_at: ActiveSupport::TimeWithZone).returns(T.untyped) }
  def add_member!(user:, role:, joined_at:)
    space_member_records.create!(user_record: user, role: role.serialize, joined_at:)
  end
end
