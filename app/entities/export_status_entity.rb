# typed: strict
# frozen_string_literal: true

class ExportStatusEntity < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :database_id, T::Wikino::DatabaseId
  const :kind, ExportStatusKind
  const :changed_at, ActiveSupport::TimeWithZone
  const :space_entity, SpaceEntity
  const :export_entity, ExportEntity
end
