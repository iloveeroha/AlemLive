sealed class ApiResult<T> {
  const ApiResult();
}

class ApiSuccess<T> extends ApiResult<T> {
  const ApiSuccess(this.data);

  final T data;
}

class ApiFailure<T> extends ApiResult<T> {
  const ApiFailure(this.message, {this.statusCode});

  final String message;
  final int? statusCode;
}
