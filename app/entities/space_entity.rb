# typed: strict
# frozen_string_literal: true

class SpaceEntity < ApplicationEntity
  sig { returns(T::Wikino::DatabaseId) }
  attr_reader :database_id

  sig { returns(String) }
  attr_reader :identifier

  sig { returns(String) }
  attr_reader :name

  sig { returns(Plan) }
  attr_reader :plan

  sig { returns(ActiveSupport::TimeWithZone) }
  attr_reader :joined_at

  sig { returns(T::Boolean) }
  attr_reader :viewer_can_update
  alias_method :viewer_can_update?, :viewer_can_update

  sig do
    params(
      database_id: T::Wikino::DatabaseId,
      identifier: String,
      name: String,
      plan: Plan,
      joined_at: ActiveSupport::TimeWithZone,
      viewer_can_update: T::Boolean
    ).void
  end
  def initialize(database_id:, identifier:, name:, plan:, joined_at:, viewer_can_update:)
    @database_id = database_id
    @identifier = identifier
    @name = name
    @plan = plan
    @joined_at = joined_at
    @viewer_can_update = viewer_can_update
  end
end
