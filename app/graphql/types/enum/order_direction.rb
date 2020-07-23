# frozen_string_literal: true

module Types
  module Enum
    class OrderDirection < Types::Enum::Base
      value "ASC", "Ascending"
      value "DESC", "Descending"
    end
  end
end
