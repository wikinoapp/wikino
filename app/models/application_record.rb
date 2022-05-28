# typed: strict
# frozen_string_literal: true

class ApplicationRecord < ActiveRecord::Base
  extend T::Sig

  self.abstract_class = true

  sig { params(ctx: GraphQL::Query::Context).returns(T::Boolean) }
  def visible_in_graphql?(ctx:)
    false
  end
end
