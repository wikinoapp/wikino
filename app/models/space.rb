# typed: strict
# frozen_string_literal: true

class SpaceEntity < ApplicationModel
  sig { returns(T::Wikino::DatabaseId) }
  attr_accessor :database_id

  sig { returns(String) }
  attr_accessor :identifier

  sig { returns(String) }
  attr_accessor :name

  sig { returns(Plan) }
  attr_accessor :plan

  sig { returns(ActiveSupport::TimeWithZone) }
  attr_accessor :joined_at

  sig { returns(T::Boolean) }
  attr_accessor :viewer_can_update
  alias_method :viewer_can_update?, :viewer_can_update

  sig { returns(T::Boolean) }
  attr_accessor :viewer_can_export
  alias_method :viewer_can_export?, :viewer_can_export
end
