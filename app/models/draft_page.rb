# typed: strict
# frozen_string_literal: true

class DraftPage < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :database_id, T::Wikino::DatabaseId
  const :modified_at, ActiveSupport::TimeWithZone
end
