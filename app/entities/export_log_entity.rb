# typed: strict
# frozen_string_literal: true

class ExportLogEntity < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :database_id, T::Wikino::DatabaseId
  const :message, String
  const :logged_at, ActiveSupport::TimeWithZone
  const :space_entity, SpaceEntity
  const :export_entity, ExportEntity
end
