# typed: strict
# frozen_string_literal: true

class ExportEntity < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :database_id, T::Wikino::DatabaseId
  const :started_by_entity, SpaceMemberEntity
  const :started_at, ActiveSupport::TimeWithZone
  const :finished_at, T.nilable(ActiveSupport::TimeWithZone)
  const :space_entity, SpaceEntity
end
