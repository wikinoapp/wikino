# typed: strict
# frozen_string_literal: true

class SpaceEntity < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :database_id, T::Wikino::DatabaseId
  const :identifier, String
  const :name, String
  const :plan, Plan
  const :joined_at, ActiveSupport::TimeWithZone
  const :viewer_can_update, T::Boolean
  const :viewer_can_export, T::Boolean

  alias_method :viewer_can_update?, :viewer_can_update
  alias_method :viewer_can_export?, :viewer_can_export
end
