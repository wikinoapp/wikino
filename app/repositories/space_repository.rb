# typed: strict
# frozen_string_literal: true

class SpaceRepository < ApplicationRepository
  # sig { params(user: User).returns(T::Array[Space]) }
  # def active_spaces(user:)
  #   user_record = UserRecord.kept.find(user.id)

  #   user_record.active_space_records.map { _1.to_model(space_viewer:) }
  # end

  sig { params(space_identifier: T.nilable(String)).returns(Space) }
  def find_by_identifier!(space_identifier)
    space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])

    build_model(space_record:)
  end

  sig { params(space_record: SpaceRecord).returns(Space) }
  def build_model(space_record:)
    Space.new(
      database_id: space_record.id,
      identifier: space_record.identifier,
      name: space_record.name,
      plan: Plan.deserialize(space_record.plan),
      joined_at: space_record.joined_at
    )
  end
end
