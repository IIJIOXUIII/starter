/**
 * Health-check shell'а — используется compose-хелсчеками и мониторингом.
 */
export default defineEventHandler(() => {
  return {
    status: 'ok',
    service: 'shell',
    time: new Date().toISOString(),
  };
});
