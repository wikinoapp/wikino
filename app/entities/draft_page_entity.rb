# typed: strict
# frozen_string_literal: true

class DraftPageEntity < ApplicationEntity
  sig { returns(T::Wikino::DatabaseId) }
  attr_reader :database_id

  sig { returns(ActiveSupport::TimeWithZone) }
  attr_reader :modified_at

  sig { returns(PageEntity) }
  attr_reader :page_entity

  sig do
    params(
      database_id: T::Wikino::DatabaseId,
      modified_at: ActiveSupport::TimeWithZone,
      page_entity: PageEntity
    ).void
  end
  def initialize(
    database_id:,
    modified_at:,
    page_entity:
  )
    @database_id = database_id
    @modified_at = modified_at
    @page_entity = page_entity
  end
end
